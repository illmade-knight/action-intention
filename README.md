# **action-intention**

This repository contains the main client application for the Action/Intention system. It is the "smart" component of the ecosystem, responsible for managing a user's local data and orchestrating the secure, federated sharing of intentions with other users.

The core architectural principle is a **local-first, federated model**. Instead of a central server, each user runs their own instance of this application, which manages their private graph of intentions, locations, and people. This approach prioritizes user privacy, data ownership, and offline capability.

## **Core Responsibilities**

* **Local Data Management:** Provides the business logic for creating and managing a user's private data (intentions, locations, people).
* **Secure Sharing Workflow:** Orchestrates the process of sharing an intention with another user. This involves:
    1. Building a self-contained SharedPayload of the relevant data.
    2. Fetching the recipient's public key from the key-service.
    3. Encrypting the payload and signing it using the crypto package.
    4. Wrapping the result in a SecureEnvelope and sending it to the routing-service.
* **Data Reconciliation:** Handles the process of receiving a SecureEnvelope. This involves:
    1. Fetching the sender's public key to verify the signature.
    2. Decrypting the SharedPayload.
    3. Using the intelligent reconciliation engine to match the incoming data with the user's local data, either by finding an exact match on a GlobalID or a fuzzy match using contextual clues.

## **System Dependencies**

This application is designed to work in concert with two external microservices and a shared types library:

1. **go-key-service**: A simple, secure microservice that acts as a public directory for user identity keys. This application communicates with it to fetch public keys for encryption and signature verification.
2. **go-routing-service**: A secure message broker that forwards encrypted SecureEnvelopes between users. This application sends outgoing payloads to the routing service and will eventually need a mechanism to pull incoming messages from it.
3. **action-intention-types**: A lightweight, shared library that defines the common data structures (SecureEnvelope, SharedPayload) used for communication across the ecosystem.

This decoupled architecture ensures that the core application logic remains separate from the complexities of key management and message transport.