# **action-intention: Development Roadmap**

## **1\. Vision & Core Principles**

The **action-intention** application is the core of a local-first, federated system that allows users to model and securely share their intended actions. The primary goal is to evolve the current prototype—which uses in-memory storage and simulated networking—into a robust, persistent, and interactive application ready for deployment.

* **Persistence:** All user data is critical and must be stored durably.
* **Connectivity:** The application must communicate with its dependent microservices over a real network.
* **User Interaction:** The application must provide a clear user interface for managing data and resolving ambiguities during data reconciliation.
* **Production Ready:** The application must be configurable, observable, and containerized for deployment.

## **2\. Current State**

The application successfully demonstrates the core end-to-end data flow in-memory.

* **Domain Logic:** The intentions, locations, and people packages contain mature business logic.
* **Sharing & Reconciliation:** The sharing and reconciliation packages correctly build payloads and perform intelligent, fuzzy matching on incoming data.
* **Cryptography:** The crypto package provides a solid foundation for hybrid encryption and digital signatures.
* **Weaknesses:** The application currently relies entirely on InMemoryStore implementations, lacks real networking clients, and has minimal error handling.

## **3\. Development Phases**

### **Phase 1: Persistence & Networking**

*Objective: Replace all in-memory components with durable, network-enabled implementations.*

1. **Persistent Storage:**
    * Replace the InMemoryStore for each domain (intentions, locations, people) with a persistent adapter.
    * **Chosen Technology:** **Firestore**. It is a managed, scalable NoSQL database that fits the document-oriented nature of the domain models.
    * **Action:** Create new storage packages (e.g., internal/storage/firestore) containing concrete Store implementations for each domain.
2. **Networking Clients:**
    * Implement real HTTP clients for communicating with the dependent microservices.
    * **Action:** Create a new internal/clients package.
        * Implement a KeyServiceClient that calls the POST /keys/{userID} and GET /keys/{userID} endpoints of the go-key-service.
        * Implement a RoutingServiceClient that calls the POST /send endpoint of the go-routing-service.
3. **Configuration Management:**
    * Externalize all hardcoded values (service URLs, ports, database project IDs) into a Config struct populated from environment variables or a configuration file.
4. **Robust Error Handling:**
    * Systematically replace all ignored errors (\_) with proper error handling, logging, and, where appropriate, returning errors up the call stack.

### **Phase 2: User Interaction & API**

*Objective: Build an API and user interface to make the application interactive.*

1. **API Layer:**
    * Create an HTTP API to expose the application's core functionality.
    * **Action:** Implement API handlers (e.g., using gin or the standard library) for:
        * Managing local data (e.g., POST /intentions, GET /locations).
        * Initiating the sharing process (POST /intentions/{id}/share).
        * Receiving incoming shared payloads.
2. **Reconciliation User Flow:**
    * The current reconciler logs MatchPossible results but takes no action. This needs to become an interactive process.
    * **Action:** When the reconciler finds a MatchPossible, the system should store this as a pending "resolution task."
    * **Action:** Create an API endpoint (e.g., GET /reconciliation/tasks) that a UI can call to present these tasks to the user, allowing them to confirm a match or create a new local entity.
3. **Incoming Message Handling:**
    * The application currently has no way to receive messages from the routing-service.
    * **Action:** Implement a mechanism to fetch incoming SecureEnvelopes. This could be a background poller that periodically calls an endpoint on the routing-service or a real-time connection (e.g., WebSocket).

### **Phase 3: Production Hardening & Deployment**

*Objective: Prepare the application for real-world deployment.*

1. **Authentication:**
    * The application needs to manage user identity.
    * **Action:** Implement a user login system (e.g., OAuth 2.0) that results in a JWT. This JWT will be sent with all authenticated requests to the dependent microservices.
2. **Observability:**
    * Integrate structured logging with zerolog throughout the application.
    * Add Prometheus metrics and a /healthz endpoint.
3. **Containerization & Deployment:**
    * Write a Dockerfile for the application.
    * Set up a CI/CD pipeline for automated testing and container builds.