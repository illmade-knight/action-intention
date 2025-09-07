// main_final_demo.go demonstrates a full, end-to-end simulation of the secure,
// federated intention sharing system.
//
// This simulation includes:
//  1. The setup of two distinct users (Alice and Bob) with their own local data stores.
//  2. The simulation of external Key and Routing services.
//  3. The generation and distribution of cryptographic keys.
//  4. The creation of a shared intention by Alice, containing a location and a person.
//  5. The complete cryptographic process:
//     - Creation of a sharable data payload (the "sub-graph").
//     - Hybrid encryption of the payload using AES-256 and RSA.
//     - Cryptographic binding of the routing information using AES-GCM's AAD feature.
//     - Digital signing of the encrypted data to ensure authenticity and integrity.
//  6. The secure transport of the message via the simulated Routing Service.
//  7. The complete reception process by Bob:
//     - Verification of the digital signature.
//     - Hybrid decryption of the payload.
//     - Reconciliation of the received data against his own local data store.
//
// To run this demo, ensure all package dependencies are available via go.mod.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/illmade-knight/action-intention/pkg/crypto"
	"github.com/illmade-knight/action-intention/pkg/intentions"
	"github.com/illmade-knight/action-intention/pkg/locations"
	"github.com/illmade-knight/action-intention/pkg/people"
	"github.com/illmade-knight/action-intention/pkg/reconciliation"
	"github.com/illmade-knight/action-intention/pkg/sharing"
	"github.com/illmade-knight/go-key-service/pkg/storage"
	"github.com/illmade-knight/go-secure-messaging/pkg/transport"
	"github.com/illmade-knight/routing-service/pkg/queue"
)

func main() {
	log.Println("--- Starting Final, Verified End-to-End Demo ---")
	ctx := context.Background()

	// --- Phase 1: System Initialization ---
	// In a real application, these services would be running as separate processes
	// on different servers. For this demo, we simulate them by instantiating their
	// in-memory storage adapters directly.
	keyServiceStore := storage.NewInMemoryStore()
	routingServiceQueue := queue.NewInMemoryQueue()

	// --- Phase 2: User Onboarding Simulation ---
	// Each user (Alice and Bob) gets their own sandboxed set of local data stores.
	// This simulates two separate client application instances.
	aliceLocationStore := locations.NewInMemoryStore()
	//alicePeopleStore := people.NewInMemoryStore()

	bobLocationStore := locations.NewInMemoryStore()
	bobPeopleStore := people.NewInMemoryStore()

	// Each user generates a public/private key pair. The private key is kept secret
	// on their local device, while the public key is uploaded to the Key Service
	// so other users can discover it.
	alicePrivateKey, alicePublicKey, _ := crypto.GenerateKeys()
	keyServiceStore.StoreKey("Alice", alicePublicKey)

	bobPrivateKey, bobPublicKey, _ := crypto.GenerateKeys()
	keyServiceStore.StoreKey("Bob", bobPublicKey)

	log.Println("✅ Setup complete. All stores and keys initialized.")

	// --- Phase 3: Seeding Local, Disparate Data ---
	log.Println("\n--- Seeding local data for each user ---")

	// Alice creates a location in her local database. The service layer correctly
	// populates both the primary `Name` field and the `Matcher.Name` field.
	aliceLocationService := locations.NewService(aliceLocationStore)
	aliceCafe, _ := aliceLocationService.AddUserLocation(ctx, "Alice", "Alice's Cafe", "Cafe")
	log.Printf("Alice's store contains: '%s' (Matcher Name: '%s')", aliceCafe.Name, aliceCafe.Matcher.Name)

	// To test the reconciliation, we create a similar entity for Bob. Bob calls his
	// location something different ("My Favorite Cafe"), but we set the underlying
	// `Matcher` data to be identical to Alice's. This simulates two users having
	// different names for the same real-world place.
	bobCafe := locations.Location{
		ID:        uuid.New(),
		Name:      "My Favorite Cafe",
		Category:  "Cafe",
		Matcher:   locations.LocationMatcher{Name: "Alice's Cafe", Category: "Cafe"},
		Type:      locations.LocationTypeUser,
		CreatedAt: time.Now(),
	}
	bobLocationStore.Add(ctx, bobCafe)
	log.Printf("Bob's store contains: '%s' (Matcher Name: '%s')", bobCafe.Name, bobCafe.Matcher.Name)

	// --- Phase 4: Alice Initiates a Secure Share with Bob ---
	log.Println("\n--- Alice is sharing an intention with Bob ---")
	intentionToShare := intentions.Intention{
		ID:     uuid.New(),
		User:   "Alice",
		Action: "Get Coffee",
		Targets: []intentions.Target{
			intentions.LocationTarget{LocationID: aliceCafe.ID},
		},
	}

	// Step 4a: Construct the sharable payload. This is a self-contained "sub-graph"
	// of all data related to the intention. The map keys are strings for JSON compatibility.
	payload := sharing.SharedPayload{
		Intention: intentionToShare,
		Locations: map[string]locations.Location{
			aliceCafe.ID.String(): aliceCafe,
		},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal payload: %v", err)
	}
	log.Println("1. Sharable payload created and serialized to JSON.")

	// Step 4b: Perform hybrid encryption using Bob's public key.
	// First, construct the Additional Authenticated Data (AAD) from the routing header.
	// This cryptographically binds the header to the payload.
	senderID := "Alice"
	recipientID := "Bob"
	additionalAuthenticatedData := []byte(senderID + ":" + recipientID)
	log.Println("2. AAD created from routing header.")

	encryptedKey, encryptedData, err := crypto.Encrypt(payloadBytes, additionalAuthenticatedData, bobPublicKey)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	log.Println("3. Payload encrypted with hybrid AES+RSA using AAD.")

	// Step 4c: Sign the *encrypted data* with Alice's private key to prove authenticity.
	signature, err := crypto.Sign(encryptedData, alicePrivateKey)
	if err != nil {
		log.Fatalf("Signing failed: %v", err)
	}
	log.Println("4. Encrypted data signed with Alice's private key.")

	// Step 4d: Assemble the final envelope and send it to the routing service.
	envelope := transport.SecureEnvelope{
		SenderID:              senderID,
		RecipientID:           recipientID,
		EncryptedData:         encryptedData,
		EncryptedSymmetricKey: encryptedKey,
		Signature:             signature,
	}
	routingServiceQueue.Enqueue(envelope)
	log.Println("5. Secure envelope sent to Routing Service.")

	// --- Phase 5: Bob Receives and Processes the Message ---
	log.Println("\n--- Bob's client is fetching and processing the message ---")

	// Step 5a: Fetch the envelope from the routing service.
	envelopes, _ := routingServiceQueue.Dequeue("Bob")
	receivedEnvelope := envelopes[0]
	log.Println("1. Secure envelope received.")

	// Step 5b: Fetch Alice's public key to verify the signature.
	alicePublicKey, _ = keyServiceStore.GetKey("Alice")
	err = crypto.Verify(receivedEnvelope.EncryptedData, receivedEnvelope.Signature, alicePublicKey)
	if err != nil {
		log.Fatalf("FATAL: Signature verification failed! Message is not authentic. %v", err)
	}
	log.Println("2. Signature is valid. Message is authentically from Alice.")

	// Step 5c: Reconstruct the AAD from the received header to verify message integrity.
	reconstructedAad := []byte(receivedEnvelope.SenderID + ":" + receivedEnvelope.RecipientID)
	log.Println("3. AAD reconstructed from received header for verification.")

	// Step 5d: Perform hybrid decryption using Bob's private key. This will fail if the
	// reconstructed AAD does not match the AAD used during encryption.
	decryptedPayloadBytes, err := crypto.Decrypt(receivedEnvelope.EncryptedSymmetricKey, receivedEnvelope.EncryptedData, reconstructedAad, bobPrivateKey)
	if err != nil {
		log.Fatalf("FATAL: Decryption failed! Tampering detected. %v", err)
	}
	log.Println("4. Payload decrypted. AAD verification passed.")

	// Step 5e: Deserialize the JSON and run the reconciliation logic.
	var receivedPayload sharing.SharedPayload
	json.Unmarshal(decryptedPayloadBytes, &receivedPayload)

	reconciler := reconciliation.NewReconciler(bobLocationStore, bobPeopleStore)
	mappingResult, _ := reconciler.ProcessPayload(ctx, receivedPayload)
	log.Println("5. Payload reconciled with Bob's local data.")

	// --- Phase 6: Final Result Verification ---
	log.Println("\n--- Final Reconciliation Result ---")
	aliceCafeID := aliceCafe.ID
	if bobMappedID, ok := mappingResult.LocationMappings[aliceCafeID]; ok {
		fmt.Printf("✅ SUCCESS: Alice's location '%s' was matched to Bob's local location (ID: %s)\n", aliceCafe.Name, bobMappedID)
	} else {
		fmt.Printf("❌ FAILED: Alice's location was NOT matched.\n")
	}
}
