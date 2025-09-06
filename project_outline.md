# **System Architecture & Design Overview**

**Last Updated:** September 5, 2025

## **1\. High-Level Vision**

This system is designed to allow users to model and share their intended actions over a period of time. The core architectural principle is a **local-first, federated model**.

Instead of a single, centralized database holding all user data, each user runs a local instance of the application. This instance manages their private data (intentions, locations, people). When a user wishes to share an intention, a secure, self-contained "sub-graph" of the relevant data is created and sent to another user via a simple routing service. The receiving user's application then intelligently reconciles this incoming data with their own local data.

This approach prioritizes user privacy, offline capability, and data ownership.

## **2\. Repository Breakdown**

The system is composed of four distinct repositories, each with a specific responsibility.

### **github.com/illmade-knight/action-intention-types**

* **Purpose:** A lightweight, shared library defining the common data structures (the "contract") used for communication between the services.
* **Key Responsibilities:**
    * Defines the SecureEnvelope, which is the object passed over the network.
    * Defines the SharedPayload, which is the unencrypted graph of data being shared.
* **Intention:** By keeping these types in a neutral repository, we prevent the client and the routing service from having a direct dependency on each other. They both depend on this shared contract, promoting loose coupling.

### **github.com/illmade-knight/key-service**

* **Purpose:** A simple, secure microservice that acts as a public directory for user identity keys.
* **Key Responsibilities:**
    * Provides an HTTP endpoint to upload a user's **public key**.
    * Provides an HTTP endpoint to fetch a user's **public key** by their ID.
* **Intention:** To completely decouple key management from application logic. This service knows nothing about intentions or routing; it only deals with public keys. It never sees, stores, or handles private keys.

### **github.com/illmade-knight/routing-service**

* **Purpose:** A secure message broker, or "post office," responsible for forwarding encrypted messages between users.
* **Key Responsibilities:**
    * Provides an HTTP endpoint to receive a SecureEnvelope.
    * Reads the unencrypted header (SenderID, RecipientID) to determine where to route the message.
    * Provides an HTTP endpoint for users to fetch any envelopes addressed to them.
* **Intention:** To act as a "dumb pipe." This service is treated as untrusted. It has **zero ability** to read the encrypted payload of the messages it handles, ensuring the confidentiality of user data.

### **github.com/illmade-knight/action-intention**

* **Purpose:** The main client application containing all the core domain logic. This is the "smart" part of the system that each user runs.
* **Key Responsibilities:**
    * Manages a user's local data (intentions, locations, people).
    * Orchestrates the sharing process: building payloads, fetching public keys, encryption, and signing.
    * Handles the receiving process: fetching messages, verification, decryption, and data reconciliation.

## **3\. Package Breakdown (action-intention repo)**

The main client application is further broken down into logical packages.

### **pkg/intentions, pkg/locations, pkg/people**

* **Intention:** These are the core **domain packages**. They define the primary data models (Intention, Location, Person, Group) and the business logic for managing them via a Service and a Store interface. They are concerned only with the user's local data. A key feature is the Matcher struct within Location and Person, which holds the necessary data for fuzzy matching during reconciliation.

### **pkg/sharing**

* **Intention:** Defines the **unencrypted graph payload**. The SharedPayload struct is a portable representation of an intention and all its related context (locations, people) *before* it is secured for transport.

### **pkg/crypto**

* **Intention:** The security layer. This package is responsible for all cryptographic operations. Its most important function is implementing the **Hybrid Encryption** scheme (AES for the large payload, RSA for the small AES key), which is critical for securely encrypting a payload of any size. It also handles the creation and verification of digital signatures to prove authenticity.

### **pkg/reconciliation**

* **Intention:** The "brains" of the federated system. The Reconciler takes an incoming SharedPayload from another user and attempts to map it to the recipient's local data. It uses a two-pronged strategy:
    1. First, it attempts to find a perfect match using a GlobalID (for public, shared entities).
    2. If that fails, it falls back to the intelligent Matcher logic to find a fuzzy match based on names, categories, and other context.

## **4\. Plan for Tomorrow: Production Refactor**

The current system successfully demonstrates the end-to-end flow but relies on in-memory stores and simulated services. The next steps are to harden this foundation for production use.

* **Persistence:** Replace all in-memory Store and Queue implementations with a real database adapter (e.g., for PostgreSQL or a local file-based DB like SQLite).
* **Networking:** Implement real HTTP clients in the action-intention app to communicate with the key-service and routing-service over the network.
* **Robust Error Handling:** Systematically handle errors that are currently being ignored with \_, providing clear feedback and fallback mechanisms.
* **Configuration Management:** Externalize hardcoded values like service URLs and ports into configuration files or environment variables.
* **User Interaction for Reconciliation:** Implement a flow for handling a MatchPossible result from the Reconciler. The system should prompt the user to confirm if the matched entity is correct or if they'd like to create a new one.
* **Authentication:** Add an authentication layer to the key-service and routing-service. When a user uploads their public key, the service must be able to verify their identity (e.g., via a JWT token from a login process).