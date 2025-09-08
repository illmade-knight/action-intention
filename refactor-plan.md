# **action-intention: Refactor Plan**

## **1\. Core Principles**

The objective is to evolve the action-intention application from a collection of in-memory domain packages into a complete, deployable service. This refactor will implement the foundational **Phase 1: Persistence & Networking** from the development roadmap.

The plan adheres to the architectural patterns established across the microservice ecosystem:

* **Decoupled Architecture:** A clean separation between the application's core logic, its external dependencies (like databases and other services), and its public-facing API.
* **Dependency Injection:** The main executable will be responsible for creating concrete dependencies (like a Firestore client) and injecting them into the application, which depends only on interfaces.
* **Testability:** The structure will support robust unit, integration, and end-to-end testing.
* **Consistency:** The final structure will mirror the go-key-service and go-routing-service for architectural consistency.

## **2\. Final Directory Structure**

The existing pkg/ directories containing the core domain logic will be preserved. We will build the application shell around them.

action-intention/  
├── cmd/  
│   └── action-intention/  
│       └── main.go              \# Assembles all dependencies and runs the application  
├── internal/  
│   ├── api/  
│   │   └── handlers.go          \# (Phase 2\) Private HTTP handlers for the UI  
│   ├── clients/  
│   │   ├── keyservice.go        \# HTTP client for the key-service  
│   │   └── routingservice.go    \# HTTP client for the routing-service  
│   └── storage/  
│       └── firestore/  
│           ├── intentions.go    \# Firestore implementation of intentions.Store  
│           ├── locations.go     \# Firestore implementation of locations.Store  
│           └── people.go        \# Firestore implementation of people.Store  
├── app/  
│   └── app.go                   \# The main application wrapper/orchestrator  
└── pkg/                         \# (Existing domain packages)  
├── crypto/  
├── intentions/  
├── locations/  
├── people/  
├── reconciliation/  
└── sharing/

## **3\. Step-by-Step Refactoring Guide**

### **Step 1: Implement Persistent Storage (Firestore)**

The first priority is to replace all InMemoryStore implementations with a durable backend.

1. **Create the Firestore Storage Package:**
    * Create the directory internal/storage/firestore/.
2. **Implement locations.Store:**
    * Create internal/storage/firestore/locations.go.
    * Implement a FirestoreStore struct that satisfies the locations.Store interface, performing CRUD operations on a locations collection in Firestore.
3. **Implement people.Store:**
    * Create internal/storage/firestore/people.go.
    * Implement a FirestoreStore struct that satisfies the people.Store interface for both Person and Group objects.
4. **Implement intentions.Store:**
    * Create internal/storage/firestore/intentions.go.
    * Implement a FirestoreStore struct that satisfies the intentions.Store interface, including the logic for Query.

### **Step 2: Implement Networking Clients**

Create clients to enable communication with the dependent microservices.

1. **Create the Clients Package:**
    * Create the directory internal/clients/.
2. **Implement KeyServiceClient:**
    * Create internal/clients/keyservice.go.
    * Implement a client with methods like GetKey(userID string) (\[\]byte, error) that make HTTP requests to the go-key-service.
3. **Implement RoutingServiceClient:**
    * Create internal/clients/routingservice.go.
    * Implement a client with a method like Send(envelope \*transport.SecureEnvelope) error that makes an HTTP POST request to the go-routing-service.

### **Step 3: Create the Main Application Wrapper**

This central component will orchestrate all the domain services and clients.

1. **Create the App Package:**
    * Create the directory app/ and the file app/app.go.
2. **Define the App Struct:**
    * The App struct will hold all the core components:
        * The domain services (intentions.Service, locations.Service, people.Service).
        * The networking clients (KeyServiceClient, RoutingServiceClient).
        * The reconciliation.Reconciler.
        * A crypto.Signer and crypto.Encrypter/Decrypter.
3. **Implement High-Level Workflows:**
    * Create methods on the App struct that orchestrate complex workflows. For example:
        * ShareIntention(ctx, intentionID, recipientID): This method will use the domain services to build the SharedPayload, use the KeyServiceClient to get the recipient's key, use the crypto package to encrypt and sign, and finally use the RoutingServiceClient to send the envelope.
        * HandleIncomingEnvelope(ctx, envelope): This method will orchestrate the reverse process of verification, decryption, and reconciliation.

### **Step 4: Create the cmd Executable**

This is the final assembly point where all concrete dependencies are created and injected.

1. **Create the Main Package:**
    * Create cmd/action-intention/main.go.
2. **Implement main() Function:**
    * **Load Configuration:** Read service URLs and the Firestore Project ID from a config file or environment variables.
    * **Initialize Clients:** Create real firestore.Client and http.Client instances.
    * **Instantiate Stores:** Create the concrete FirestoreStore adapters from internal/storage/firestore/.
    * **Instantiate Services:** Create the domain Service instances (intentions, locations, people), injecting the Firestore stores into them.
    * **Instantiate App Wrapper:** Create the main App object, injecting all the services and clients.
    * **Start the Application:** (Future) Start the API server from Phase 2\. For now, this could run a CLI or a test function.