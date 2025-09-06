// FILE: main_final_demo.go (Corrected & Final)

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

	// ... (Setup is the same) ...
	keyServiceStore := storage.NewInMemoryStore()
	routingServiceQueue := queue.NewInMemoryQueue()
	aliceLocationStore := locations.NewInMemoryStore()
	bobLocationStore := locations.NewInMemoryStore()
	bobPeopleStore := people.NewInMemoryStore()
	alicePrivKey, alicePubKey, _ := crypto.GenerateKeys()
	keyServiceStore.StoreKey("Alice", alicePubKey)
	bobPrivKey, bobPubKey, _ := crypto.GenerateKeys()
	keyServiceStore.StoreKey("Bob", bobPubKey)
	log.Println("✅ Setup complete.")

	// ... (Seeding data is the same) ...
	log.Println("\n--- Seeding local data for each user ---")
	aliceLocationService := locations.NewService(aliceLocationStore)
	aliceCafe, _ := aliceLocationService.AddUserLocation(ctx, "Alice", "Alice's Cafe", "Cafe")
	bobCafe := locations.Location{ID: uuid.New(), Name: "My Favorite Cafe", Category: "Cafe", Matcher: locations.LocationMatcher{Name: "Alice's Cafe", Category: "Cafe"}, Type: locations.LocationTypeUser, CreatedAt: time.Now()}
	bobLocationStore.Add(ctx, bobCafe)

	log.Println("\n--- Alice is sharing an intention with Bob ---")
	intentionToShare := intentions.Intention{ID: uuid.New(), User: "Alice", Action: "Get Coffee"}
	intentionToShare.Targets = []intentions.Target{intentions.LocationTarget{LocationID: aliceCafe.ID}}
	payload := sharing.SharedPayload{
		Intention: intentionToShare,
		Locations: map[string]locations.Location{aliceCafe.ID.String(): aliceCafe},
	}
	payloadBytes, _ := json.Marshal(payload)
	log.Println("1. Payload created.")

	// --- HYBRID ENCRYPTION FLOW ---
	encryptedKey, encryptedData, err := crypto.Encrypt(payloadBytes, bobPubKey)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}
	log.Println("2. Payload encrypted with hybrid AES+RSA.")

	// Sign the encrypted DATA, not the key.
	signature, _ := crypto.Sign(encryptedData, alicePrivKey)
	log.Println("3. Encrypted data signed with Alice's private key.")

	envelope := transport.SecureEnvelope{
		SenderID:              "Alice",
		RecipientID:           "Bob",
		EncryptedData:         encryptedData,
		EncryptedSymmetricKey: encryptedKey,
		Signature:             signature,
	}
	routingServiceQueue.Enqueue(envelope)
	log.Println("4. Secure envelope sent to Routing Service.")

	// --- BOB'S DECRYPTION FLOW ---
	log.Println("\n--- Bob's client is fetching and processing the message ---")
	envelopes, _ := routingServiceQueue.Dequeue("Bob")
	receivedEnvelope := envelopes[0]
	log.Println("1. Secure envelope received.")

	alicePubKey, _ = keyServiceStore.GetKey("Alice")
	err = crypto.Verify(receivedEnvelope.EncryptedData, receivedEnvelope.Signature, alicePubKey)
	if err != nil {
		log.Fatalf("FATAL: Signature verification failed! %v", err)
	}
	log.Println("2. Signature is valid.")

	decryptedPayloadBytes, err := crypto.Decrypt(receivedEnvelope.EncryptedSymmetricKey, receivedEnvelope.EncryptedData, bobPrivKey)
	if err != nil {
		log.Fatalf("FATAL: Decryption failed! %v", err)
	}
	log.Println("3. Payload decrypted with Bob's private key.")

	var receivedPayload sharing.SharedPayload
	json.Unmarshal(decryptedPayloadBytes, &receivedPayload)

	reconciler := reconciliation.NewReconciler(bobLocationStore, bobPeopleStore)
	mappingResult, _ := reconciler.ProcessPayload(ctx, receivedPayload)
	log.Println("4. Payload reconciled.")

	// --- FINAL RESULT ---
	log.Println("\n--- Final Reconciliation Result ---")
	aliceCafeID := aliceCafe.ID
	if bobMappedID, ok := mappingResult.LocationMappings[aliceCafeID]; ok {
		fmt.Printf("✅ SUCCESS: Alice's location '%s' was matched to Bob's local location (ID: %s)\n", aliceCafe.Name, bobMappedID)
	} else {
		fmt.Printf("❌ FAILED: Alice's location was NOT matched.\n")
	}
}
