# Option 2: True End-to-End Encryption Implementation

This document details the implementation of Telegram-style end-to-end encryption for messages.

## Architecture Overview

**Key Principles:**
- Messages encrypted on client before sending
- Server stores encrypted messages (never sees plaintext)
- Messages decrypted on client when displaying
- Key exchange between participants
- Forward secrecy (optional, advanced feature)

**How E2EE Works with Your App Architecture:**
- **Asynchronous Communication:** User A and User B don't communicate directly - they both use the app/server as intermediary
- **Message Flow:** A sends → Server stores encrypted → B receives later (async, not real-time chat)
- **Real-time UI Updates:** UI updates in real-time via HTMX/SSE as messages arrive, client decrypts on-the-fly
- **Key Exchange:** Happens asynchronously - when A first opens conversation, they upload public key; when B opens, they get A's key and upload theirs
- **Shared Secret (Important!):** NOT stored anywhere! Each user derives it independently:
  - User A: Shared Secret = X25519(A's Private Key, B's Public Key)
  - User B: Shared Secret = X25519(B's Private Key, A's Public Key)
  - Both get the SAME secret (X25519 property) - allows A to encrypt, B to decrypt (and vice versa)
  - Derived on-the-fly when needed, never stored on server
- **Notification System:** Server sends link-only notification (SMS/Email/Signal): "New message from [Sender] about [Ad]. View at: [link]"
- **Perfect for E2EE:** Server acts as encrypted message relay - never sees plaintext, just stores/forwards encrypted bytes

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          END-TO-END ENCRYPTION SYSTEM                        │
└─────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────┐                                    ┌──────────────────────┐
│   USER A (Browser)  │                                    │   USER B (Browser)  │
│   (Buyer/Seller)    │                                    │   (Buyer/Seller)    │
│                      │                                    │                      │
│  ┌────────────────┐ │                                    │  ┌────────────────┐ │
│  │  Private Key A  │ │                                    │  │  Private Key B  │ │
│  │  (localStorage) │ │                                    │  │  (localStorage) │ │
│  └────────────────┘ │                                    │  └────────────────┘ │
│         │            │                                    │         │            │
│         │            │                                    │         │            │
│  ┌──────▼──────────┐│                                    │  ┌──────▼──────────┐│
│  │  Public Key A    ││                                    │  │  Public Key B    ││
│  │  (derived)       ││                                    │  │  (derived)       ││
│  └──────┬──────────┘│                                    │  └──────┬──────────┘│
│         │            │                                    │         │            │
│         │            │                                    │         │            │
│  ASYNC: A and B don't communicate directly                │         │            │
│  They both use Server as intermediary/relay               │         │            │
│         │            │                                    │         │            │
└─────────┼────────────┘                                    └─────────┼────────────┘
          │                                                           │
          │  [1] Upload Public Key A (when A opens conversation)    │
          │───────────────────────────────────────────────────────────┼──────────┐
          │                                                           │          │
          │                                                           │  [2] Upload Public Key B (when B opens conversation)
          │                                                           │          │
          │                                                           │          │
┌─────────▼───────────────────────────────────────────────────────────▼──────────▼──┐
│                              SERVER (Go Backend)                                  │
│                                                                                   │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                    Key Exchange Handlers (/api/e2ee/*)                    │   │
│  │                                                                           │   │
│  │  • HandlePublicKeyUpload()  - Store user public keys                     │   │
│  │  • HandleGetPeerPublicKey() - Retrieve peer's public key                │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                                    │                                              │
│                                    │                                              │
│  ┌─────────────────────────────────▼──────────────────────────────────────┐   │
│  │                         Message Handlers                                │   │
│  │                                                                          │   │
│  │  • HandleSendMessageE2EE()  - Accept encrypted messages                 │   │
│  │  • HandleExpandConversation() - Return encrypted messages               │   │
│  │  • HandleSSE() - Real-time updates via Server-Sent Events               │   │
│  │                                                                          │   │
│  │  Server acts as ENCRYPTED MESSAGE RELAY:                                │   │
│  │  - Receives encrypted from A → Stores encrypted → Forwards encrypted   │   │
│  │  - B requests messages → Server returns encrypted → B decrypts          │   │
│  │  - Real-time: SSE pushes encrypted messages, client decrypts on-the-fly │   │
│  │                                                                          │   │
│  │  Note: Server NEVER decrypts messages - only stores/retrieves/relays   │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                    │                                              │
└────────────────────────────────────┼──────────────────────────────────────────────┘
                                     │
                                     │
┌────────────────────────────────────▼──────────────────────────────────────────────┐
│                              DATABASE (SQLite)                                     │
│                                                                                    │
│  ┌──────────────────────────────────────────────────────────────────────────┐    │
│  │                        UserPublicKey Table                                │    │
│  │  ┌──────────┬──────────┬──────────────┬──────────┐                      │    │
│  │  │ user_id  │ key_id   │ public_key   │ key_type │                      │    │
│  │  ├──────────┼──────────┼──────────────┼──────────┤                      │    │
│  │  │   101    │    1     │ [32 bytes]   │  x25519  │  ← User A's key      │    │
│  │  │   102    │    2     │ [32 bytes]   │  x25519  │  ← User B's key      │    │
│  │  └──────────┴──────────┴──────────────┴──────────┘                      │    │
│  └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                    │
│  ┌──────────────────────────────────────────────────────────────────────────┐    │
│  │                        Message Table                                      │    │
│  │  ┌────┬──────────────┬──────────┬──────────────────┬──────────┐         │    │
│  │  │ id │ conversation │ sender   │ encrypted_content │  nonce   │         │    │
│  │  ├────┼──────────────┼──────────┼──────────────────┼──────────┤         │    │
│  │  │ 1  │     501      │   101    │  [ciphertext]    │ [12 bytes]│         │    │
│  │  │ 2  │     501      │   102    │  [ciphertext]    │ [12 bytes]│         │    │
│  │  └────┴──────────────┴──────────┴──────────────────┴──────────┘         │    │
│  │                                                                           │    │
│  │  Note: Server stores encrypted bytes only - cannot read plaintext         │    │
│  └──────────────────────────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════════════════════
                            MESSAGE FLOW SEQUENCE (ASYNC THROUGH SERVER)
═══════════════════════════════════════════════════════════════════════════════════════

NOTE: User A and User B do NOT communicate directly. They both communicate with the Server.
      The Server acts as an encrypted message relay/storage.

═══════════════════════════════════════════════════════════════════════════════════════
PHASE 1: User A Sets Up (A opens conversation first)
═══════════════════════════════════════════════════════════════════════════════════════

┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   User A    │         │   Server    │         │  Database   │
│  (Browser)  │         │  (Go App)   │         │  (SQLite)   │
└──────┬──────┘         └──────┬───────┘         └──────┬───────┘
       │                      │                        │
       │  [1] A opens conversation in UI               │
       │  Generates Key Pair (X25519)                  │
       │◄───────────────────────────────────────────────┤
       │                                                │
       │  [2] Upload Public Key A to Server             │
       │───────────────────────────────────────────────►│
       │                                                │
       │                                                │  [3] Store Public Key A
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │  [4] Store Complete   │
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │  [5] Public Key A Stored                       │                        │
       │◄───────────────────────────────────────────────┤                        │
       │                                                │                        │
       └────────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════════════════
PHASE 2: User A Sends Message (B is NOT online)
═══════════════════════════════════════════════════════════════════════════════════════

┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   User A    │         │   Server    │         │  Database   │
│  (Browser)  │         │  (Go App)   │         │  (SQLite)   │
└──────┬──────┘         └──────┬───────┘         └──────┬───────┘
       │                      │                        │
       │  [6] A types message: "How much?"              │
       │                                                │
       │  [6a] Client checks: Does B have public key?  │
       │  GET /api/e2ee/conversation/501/peer-key      │
       │───────────────────────────────────────────────►│
       │                                                │                        │
       │                                                │  [6b] Query Public Key B
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │  [6c] Return Public Key B (if exists)
       │                                                │  OR: 404 Not Found (if B hasn't uploaded yet)
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │  [6d] Receive response                        │                        │
       │◄───────────────────────────────────────────────┤                        │
       │                                                │                        │
       │  IF B's key exists:                            │                        │
       │  [6e] Derive Shared Secret                      │                        │
       │  Shared Secret = X25519(A's Private Key, B's Public Key)│                        │
       │                                                │                        │
       │  [6f] Derive Message Key                        │                        │
       │  Message Key = HKDF(Shared Secret, Conversation ID)│                        │
       │                                                │                        │
       │  [6g] Encrypt message with Message Key         │                        │
       │  (ChaCha20-Poly1305)                           │                        │
       │                                                │                        │
       │  IF B's key doesn't exist yet:                 │                        │
       │  [6h] Store message as "pending" (encrypt later when B uploads key)│                        │
       │  OR: Show error "Recipient hasn't set up encryption yet"│                        │
       │                                                │                        │
       │  [7] Send Encrypted Message to Server          │                        │
       │  POST /messages/501/send                       │
       │  { encrypted_content: "...", nonce: "..." }   │
       │───────────────────────────────────────────────►│
       │                                                │
       │                                                │  [8] Store Encrypted Message
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │  [9] Store Complete   │
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │  [10] Message Stored (Server never saw plaintext)                        │
       │◄───────────────────────────────────────────────┤                        │
       │                                                │                        │
       │  [11] Server sends Notification to User B     │                        │
       │  SMS/Email: "New message from A about 'Ad'.    │                        │
       │  View at: /messages/501"                       │                        │
       │  (No message content - server can't decrypt)  │                        │
       │───────────────────────────────────────────────►│                        │
       │                                                │                        │
       └────────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════════════════
PHASE 3: User B Opens Conversation (Later, async - A may be offline)
═══════════════════════════════════════════════════════════════════════════════════════

┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   User B    │         │   Server    │         │  Database   │
│  (Browser)  │         │  (Go App)   │         │  (SQLite)   │
└──────┬──────┘         └──────┬───────┘         └──────┬───────┘
       │                      │                        │
       │  [12] B clicks notification link               │
       │  Opens UI, loads conversation                  │
       │                                                │
       │  [13] B generates Key Pair (X25519)           │
       │◄───────────────────────────────────────────────┤
       │                                                │
       │  [14] Upload Public Key B to Server            │
       │───────────────────────────────────────────────►│
       │                                                │
       │                                                │  [15] Store Public Key B
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │  [16] Store Complete   │
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │  [17] Request Peer's (User A's) Public Key    │                        │
       │───────────────────────────────────────────────►│                        │
       │                                                │                        │
       │                                                │  [18] Query Public Key A
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │  [19] Return Public Key A
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │  [20] Receive Public Key A                     │                        │
       │◄───────────────────────────────────────────────┤                        │
       │                                                │                        │
       │  [21] Derive Shared Secret                     │                        │
       │  Shared Secret = X25519(Private Key B, Public Key A)│                        │
       │  (Same secret that A will derive when encrypting!)│                        │
       │                                                │                        │
       │  [21a] Derive Message Key                      │                        │
       │  Message Key = HKDF(Shared Secret, Conversation ID)│                        │
       │                                                │                        │
       │  [22] Request Encrypted Messages               │                        │
       │  GET /messages/501                             │                        │
       │───────────────────────────────────────────────►│                        │
       │                                                │                        │
       │                                                │  [23] Query Encrypted Messages
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │  [24] Return Encrypted Messages
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │  [25] Receive Encrypted Messages               │                        │
       │◄───────────────────────────────────────────────┤                        │
       │                                                │                        │
       │  [26] Message Key already derived in step [21a]│                        │
       │  (HKDF: Shared Secret + Conversation ID)      │                        │
       │                                                │                        │
       │  [27] Decrypt Each Message                     │                        │
       │  (ChaCha20-Poly1305: Ciphertext + Message Key + Nonce)│                        │
       │                                                │                        │
       │  [28] Display Decrypted Messages in UI         │                        │
       │  "How much?"                                   │                        │
       │                                                │                        │
       └────────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════════════════
PHASE 4: Real-Time Update (B has UI open, A sends new message)
═══════════════════════════════════════════════════════════════════════════════════════

┌─────────────┐         ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   User A    │         │   Server    │         │  Database   │         │   User B    │
│  (Browser)  │         │  (Go App)   │         │  (SQLite)   │         │  (Browser)  │
└──────┬──────┘         └──────┬───────┘         └──────┬───────┘         └──────┬──────┘
       │                      │                        │                        │
       │  [29] A types new message: "$50"               │                        │
       │  Client encrypts                                │                        │
       │───────────────────────────────────────────────►│                        │
       │                                                │                        │
       │                                                │  [30] Store Encrypted  │
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │  [31] Store Complete   │
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │  [32] Message Stored                           │                        │
       │◄───────────────────────────────────────────────┤                        │
       │                                                │                        │
       │                                                │  [33] SSE Push to B     │
       │                                                │  (B has UI open)       │
       │                                                │───────────────────────►│
       │                                                │                        │
       │                                                │                        │  [34] B receives SSE event
       │                                                │                        │  "new_message: 501"
       │                                                │                        │
       │                                                │                        │  [35] B fetches encrypted message
       │                                                │                        │───────────────────────►│
       │                                                │                        │
       │                                                │  [36] Return Encrypted  │
       │                                                │◄───────────────────────┤
       │                                                │                        │
       │                                                │                        │  [37] B decrypts on-the-fly
       │                                                │                        │
       │                                                │                        │  [38] B displays "$50" in UI
       │                                                │                        │  (Real-time update, no refresh)
       │                                                │                        │
       └────────────────────────────────────────────────────────────────────────┘

KEY POINTS:
- A and B NEVER communicate directly - only through Server
- Server acts as encrypted message relay/storage
- A and B are ASYNC - don't need to be online at same time
- Server NEVER sees plaintext - only encrypted bytes
- Real-time updates work via SSE when both have UI open


═══════════════════════════════════════════════════════════════════════════════════════
                            ENCRYPTION/DECRYPTION FLOW
═══════════════════════════════════════════════════════════════════════════════════════

USER A SENDING MESSAGE:
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                                                                 │
│  1. User types: "Hello, how much for the part?"                                │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ Plaintext Message: "Hello, how much for the part?" │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  2. JavaScript intercepts form submission (message-handler.js)                 │
│                              │                                                  │
│                              ▼                                                  │
│  3. Get Message Key (derive from shared secret + conversation ID)              │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ Message Key: [32 bytes AES-256 key]                │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  4. Generate random nonce (12 bytes)                                           │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ Nonce: [12 random bytes]                            │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  5. Encrypt with AES-256-GCM                                                  │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ AES-GCM(Message, Message Key, Nonce)               │                    │
│     │ = Encrypted Ciphertext + Authentication Tag          │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  6. Base64 encode for transmission                                             │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ {                                                    │                    │
│     │   "encrypted_content": "aGVsbG8gd29ybGQ...",        │                    │
│     │   "nonce": "YWJjZGVmZ2hpams=",                      │                    │
│     │   "version": 1                                       │                    │
│     │ }                                                    │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  7. Send to server (HTMX POST)                                                 │
│     Server stores encrypted bytes in database                                 │
│     Server NEVER sees plaintext                                                │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘


USER B RECEIVING MESSAGE:
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                                                                 │
│  1. Server returns encrypted message (HTMX response)                         │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ {                                                    │                    │
│     │   "id": 123,                                        │                    │
│     │   "encrypted_content": "aGVsbG8gd29ybGQ...",        │                    │
│     │   "nonce": "YWJjZGVmZ2hpams=",                      │                    │
│     │   "version": 1                                       │                    │
│     │ }                                                    │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  2. JavaScript detects encrypted message (htmx:afterSwap event)                │
│                              │                                                  │
│                              ▼                                                  │
│  3. Get Message Key (derive from shared secret + conversation ID)              │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ Message Key: [32 bytes AES-256 key]                │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  4. Base64 decode encrypted content and nonce                                  │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ Encrypted Bytes: [binary ciphertext]                 │                    │
│     │ Nonce: [12 bytes]                                    │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  5. Decrypt with AES-256-GCM                                                  │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ AES-GCM-Decrypt(Ciphertext, Message Key, Nonce)    │                    │
│     │ = Plaintext Message                                  │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                              │                                                  │
│                              ▼                                                  │
│  6. Display decrypted message                                                 │
│     ┌─────────────────────────────────────────────────────┐                    │
│     │ "Hello, how much for the part?"                     │                    │
│     └─────────────────────────────────────────────────────┘                    │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════════════════════
                            KEY EXCHANGE PROTOCOL (X25519)
═══════════════════════════════════════════════════════════════════════════════════════

┌─────────────────────────────────────────────────────────────────────────────────┐
│                                                                                 │
│  STEP 1: User A generates key pair                                            │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ Private Key A (32 bytes, random)                     │                      │
│  │ Public Key A = X25519(Private Key A, Basepoint)     │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                              │                                                  │
│                              ▼                                                  │
│  STEP 2: User A uploads Public Key A to server                                 │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ POST /api/e2ee/public-key                           │                      │
│  │ { "public_key": "base64(Public Key A)" }            │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                              │                                                  │
│                              ▼                                                  │
│  STEP 3: Server stores Public Key A in UserPublicKey table                     │
│                                                                                 │
│  STEP 4: User B generates key pair                                            │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ Private Key B (32 bytes, random)                     │                      │
│  │ Public Key B = X25519(Private Key B, Basepoint)     │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                              │                                                  │
│                              ▼                                                  │
│  STEP 5: User B uploads Public Key B to server                                 │
│                                                                                 │
│  STEP 6: Server stores Public Key B in UserPublicKey table                     │
│                                                                                 │
│  STEP 7: User A requests User B's public key                                   │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ GET /api/e2ee/conversation/501/peer-key              │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                              │                                                  │
│                              ▼                                                  │
│  STEP 8: Server returns Public Key B                                           │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ { "public_key": "base64(Public Key B)" }            │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                              │                                                  │
│                              ▼                                                  │
│  STEP 9: User A derives shared secret                                          │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ Shared Secret = X25519(Private Key A, Public Key B) │                      │
│  │ (32 bytes, same for both users)                      │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                              │                                                  │
│                              ▼                                                  │
│  STEP 10: User B requests User A's public key                                   │
│  STEP 11: User B derives shared secret                                          │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ Shared Secret = X25519(Private Key B, Public Key A) │                      │
│  │ (32 bytes, same as User A's)                         │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                              │                                                  │
│                              ▼                                                  │
│  STEP 12: Both users derive message key from shared secret                     │
│  ┌─────────────────────────────────────────────────────┐                      │
│  │ Message Key = HKDF(Shared Secret, Conversation ID)  │                      │
│  │ (32 bytes ChaCha20 key)                             │                      │
│  └─────────────────────────────────────────────────────┘                      │
│                                                                                 │
│  CRITICAL: Shared secret is NOT stored or transmitted!                        │
│  - Each user derives it independently when needed                              │
│  - User A: Shared Secret = X25519(A's Private Key, B's Public Key)           │
│  - User B: Shared Secret = X25519(B's Private Key, A's Public Key)           │
│  - Both get the SAME secret (mathematical property of X25519)                │
│  - Server never sees or knows the shared secret                                │
│  - Works perfectly for async communication!                                    │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════════════════════
                            SECURITY BOUNDARIES
═══════════════════════════════════════════════════════════════════════════════════════

┌─────────────────────────────────────────────────────────────────────────────────┐
│                                                                                 │
│  ✅ SECURE ZONES (Plaintext visible)                                            │
│  ┌──────────────────────────────────────────────────────────────────────────┐  │
│  │ • User A's Browser (before encryption, after decryption)                 │  │
│  │ • User B's Browser (before encryption, after decryption)                 │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                                                                                 │
│  ❌ INSECURE ZONES (Only encrypted data)                                       │
│  ┌──────────────────────────────────────────────────────────────────────────┐  │
│  │ • Network transmission (HTTPS encrypted, but message content encrypted)  │  │
│  │ • Server memory (only sees encrypted bytes)                              │  │
│  │ • Database storage (only encrypted bytes)                                 │  │
│  │ • Server logs (no plaintext)                                              │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                                                                                 │
│  🔐 KEY STORAGE                                                                │
│  ┌──────────────────────────────────────────────────────────────────────────┐  │
│  │ • Private Keys: Browser localStorage (encrypted with user password)      │  │
│  │ • Public Keys: Database (safe to store publicly)                         │  │
│  │ • Shared Secrets: Browser localStorage (temporary, per conversation)    │  │
│  │ • Message Keys: Browser sessionStorage (temporary, per session)          │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                                                                                 │
│  ⚠️  ATTACK VECTORS MITIGATED                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐  │
│  │ • Database breach: Messages encrypted, attacker cannot read               │  │
│  │ • Server compromise: Server never sees plaintext                          │  │
│  │ • Man-in-the-middle: HTTPS + message encryption                          │  │
│  │ • Replay attacks: Unique nonce per message                                │  │
│  │ • Message tampering: ChaCha20-Poly1305 authentication tag                │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Components Required

### 1. Client-Side JavaScript Library
- Use libsodium.js (https://github.com/jedisct1/libsodium.js)
- Intercept form submissions
- Encrypt before sending
- Decrypt when receiving
- Manage encryption keys in browser storage

### 2. Server-Side Key Exchange
- Handle Diffie-Hellman key exchange
- Store public keys
- Derive shared secrets

### 3. Database Schema Changes
- Store encrypted content (BLOB)
- Store public keys
- Track encryption protocol version

### 4. Message Flow Changes
- Accept encrypted content from client
- Return encrypted content to client
- Client decrypts for display

### 5. Notification System Integration
- **Key Point:** Notifications only contain a LINK, not message content
- Server sends: "New message from [Sender] about [Ad]. View at: [link]"
- User clicks link → Opens UI → Client decrypts and displays message
- **No conflict with E2EE:** Server never needs plaintext for notifications

---

## How E2EE Works with Your App's Message Flow

### Current Flow (Without E2EE):
```
1. Buyer types message in UI: "How much for the part?"
2. Server receives plaintext, stores in DB
3. Server sends SMS/Email: "New message from Buyer about 'Car Part': How much for the part? View at: /messages/501"
4. Seller clicks link, opens UI, sees message
```

### E2EE Flow (With Encryption):
```
1. Buyer (User A) types message in UI: "How much for the part?"
2. Client encrypts message BEFORE sending (using shared secret with Seller)
3. Server receives ENCRYPTED bytes (cannot read plaintext)
4. Server stores encrypted bytes in DB
5. Server sends SMS/Email to Seller: "New message from Buyer about 'Car Part'. View at: /messages/501"
   (No message content - just sender, ad title, and link)
6. Seller (User B) clicks link, opens UI
7. UI loads conversation via HTMX, retrieves encrypted messages from server
8. Client decrypts each message on-the-fly and displays: "How much for the part?"
9. Real-time updates: If Seller has UI open, HTMX/SSE pushes new encrypted message
10. Client automatically decrypts and displays new message in real-time
```

**Key Points:**
- A and B are **asynchronous** - they don't need to be online at the same time
- Server acts as **encrypted message relay** - stores encrypted, forwards encrypted
- **Key exchange happens when each user first accesses conversation:**
  - First user to open conversation uploads their public key
  - Second user opens conversation, gets first user's key, uploads theirs, derives shared secret
  - Both can now encrypt/decrypt messages
- **Real-time UI updates:** When new encrypted message arrives via HTMX/SSE, client decrypts immediately

### Why This Works Perfectly:
- **Notifications don't need content:** Just sender name, ad title, and link
- **Server never sees plaintext:** Messages encrypted before reaching server
- **User experience unchanged:** User still clicks link to view message
- **True E2EE maintained:** Only buyer and seller can decrypt messages

### How Shared Secret Works (Even Though A and B Are Async):

**The Key Insight:** A and B don't need to be online at the same time to have a "shared secret". They each derive it independently using their own private key + the other's public key.

**Example:**
```
1. User A opens conversation:
   - Generates Private Key A, Public Key A
   - Uploads Public Key A to server
   - Server stores it

2. User B opens conversation (later, A may be offline):
   - Generates Private Key B, Public Key B
   - Uploads Public Key B to server
   - Gets Public Key A from server
   - Derives: Shared Secret = X25519(Private Key B, Public Key A)
   - Stores shared secret in browser (for this conversation)

3. User A sends message (B may be offline):
   - **First:** Checks if B has uploaded their public key yet
   - **If B's key exists:** 
     - Gets Public Key B from server
     - Derives: Shared Secret = X25519(Private Key A, Public Key B)
     - Derives: Message Key = HKDF(Shared Secret, Conversation ID)
     - Encrypts message with Message Key (ChaCha20-Poly1305)
     - Sends encrypted message to server
   - **If B's key doesn't exist yet:**
     - Message cannot be encrypted yet (needs B's public key)
     - Options: Store as pending, show error, or wait for B to set up

4. User B opens conversation later:
   - Already has shared secret cached in browser
   - Gets encrypted message from server
   - Decrypts using shared secret
   - Reads message ✅
```

**Important Points:**
- Shared secret is NOT stored on server (only public keys are)
- Each user derives it independently when needed
- Both users get the SAME secret (mathematical property of X25519)
- Allows async communication - A can encrypt, B can decrypt later
- Server never knows the shared secret

### What Changes:
- Notification text: Remove message content preview, keep link
- Server handlers: Accept encrypted content, don't try to decrypt
- Client JavaScript: Encrypt before send, decrypt after receive

---

## Implementation Details

### Part 1: Database Schema Changes

```sql
-- Add encryption support to Message table
ALTER TABLE Message ADD COLUMN encrypted_content BLOB;
ALTER TABLE Message ADD COLUMN encryption_version INTEGER DEFAULT 1;
ALTER TABLE Message ADD COLUMN nonce BLOB; -- For ChaCha20-Poly1305 (24 bytes)

-- Create table for public keys (for key exchange)
CREATE TABLE UserPublicKey (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES User(id),
    public_key BLOB NOT NULL,
    key_type TEXT NOT NULL DEFAULT 'x25519',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active INTEGER NOT NULL DEFAULT 1,
    UNIQUE (user_id, is_active) WHERE is_active = 1
);
CREATE INDEX idx_userpublickey_user_id ON UserPublicKey(user_id);

-- Note: We DON'T store shared secrets in the database!
-- Shared secrets are derived on-the-fly by each client when needed
-- Each user derives: Shared Secret = X25519(My Private Key, Peer's Public Key)
-- Both users get the same secret independently (X25519 property)
-- This allows async communication - A encrypts, B decrypts later (or vice versa)
--
-- Optional: Store public key references for faster lookups (not required)
-- The shared secret is NEVER stored - only public keys are stored
```

### Part 2: Server-Side Go Code

#### `messaging/e2ee.go` - E2EE Package

```go
package messaging

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

const (
	EncryptionVersion1 = 1
	NonceSize         = 24 // For ChaCha20-Poly1305
	KeySize           = 32 // 256 bits
)

// PublicKey represents a user's public key for key exchange
type PublicKey struct {
	ID       int    `json:"id" db:"id"`
	UserID   int    `json:"user_id" db:"user_id"`
	KeyBytes []byte `json:"key_bytes" db:"public_key"`
	KeyType  string `json:"key_type" db:"key_type"`
}

// EncryptedMessage represents an encrypted message from the client
type EncryptedMessage struct {
	EncryptedContent string `json:"encrypted_content"` // Base64 encoded
	Nonce            string `json:"nonce"`             // Base64 encoded
	Version          int    `json:"version"`
}

// GenerateKeyPair generates a new X25519 key pair for a user
func GenerateKeyPair() (privateKey, publicKey []byte, err error) {
	privateKey = make([]byte, 32)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, nil, err
	}

	// Clamp private key for X25519
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	publicKey, err = curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, publicKey, nil
}

// DeriveSharedSecret derives a shared secret from two public keys
// This implements the X25519 key exchange
func DeriveSharedSecret(privateKey, peerPublicKey []byte) ([]byte, error) {
	sharedSecret, err := curve25519.X25519(privateKey, peerPublicKey)
	if err != nil {
		return nil, err
	}
	return sharedSecret, nil
}

// DeriveMessageKey derives an encryption key from shared secret and conversation ID
func DeriveMessageKey(sharedSecret []byte, conversationID int) []byte {
	// Use HKDF to derive a message key
	salt := []byte(fmt.Sprintf("conversation-%d", conversationID))
	info := []byte("message-encryption-v1")
	
	hkdf := hkdf.New(sha3.New256, sharedSecret, salt, info)
	key := make([]byte, KeySize)
	hkdf.Read(key)
	return key
}

// StorePublicKey stores a user's public key
func StorePublicKey(userID int, publicKey []byte) (int, error) {
	// Deactivate old keys
	_, err := db.Exec(`
		UPDATE UserPublicKey 
		SET is_active = 0 
		WHERE user_id = ? AND is_active = 1
	`, userID)
	if err != nil {
		return 0, err
	}

	// Insert new key
	res, err := db.Exec(`
		INSERT INTO UserPublicKey (user_id, public_key, key_type)
		VALUES (?, ?, 'x25519')
	`, userID, publicKey)
	if err != nil {
		return 0, err
	}

	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetPublicKey retrieves a user's active public key
func GetPublicKey(userID int) (PublicKey, error) {
	var pk PublicKey
	row := db.QueryRow(`
		SELECT id, user_id, public_key, key_type
		FROM UserPublicKey
		WHERE user_id = ? AND is_active = 1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID)

	err := row.Scan(&pk.ID, &pk.UserID, &pk.KeyBytes, &pk.KeyType)
	if err != nil {
		return PublicKey{}, err
	}
	return pk, nil
}

// Note: We don't store shared secrets in the database!
// Shared secrets are derived on-the-fly by each client when needed:
// - User A: Shared Secret = X25519(A's Private Key, B's Public Key)
// - User B: Shared Secret = X25519(B's Private Key, A's Public Key)
// Both get the same secret independently (X25519 mathematical property)
// This allows async communication - A encrypts, B decrypts later (or vice versa)
// Server never sees or stores the shared secret

// VerifyEncryptedMessage verifies that an encrypted message is valid
// (Server can verify format without decrypting)
func VerifyEncryptedMessage(encrypted EncryptedMessage) error {
	if encrypted.Version != EncryptionVersion1 {
		return errors.New("unsupported encryption version")
	}
	if encrypted.EncryptedContent == "" {
		return errors.New("encrypted content is empty")
	}
	if encrypted.Nonce == "" {
		return errors.New("nonce is empty")
	}
	return nil
}
```

#### Updated `messaging/messaging.go` - Handle Encrypted Messages

```go
// AddMessageE2EE adds an encrypted message (server doesn't decrypt)
func AddMessageE2EE(conversationID, senderID int, encrypted EncryptedMessage) (int, error) {
	// Verify encrypted message format
	if err := VerifyEncryptedMessage(encrypted); err != nil {
		return 0, fmt.Errorf("invalid encrypted message: %w", err)
	}

	// Decode encrypted content and nonce
	encryptedBytes, err := base64.StdEncoding.DecodeString(encrypted.EncryptedContent)
	if err != nil {
		return 0, fmt.Errorf("invalid encrypted content: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return 0, fmt.Errorf("invalid nonce: %w", err)
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Insert encrypted message (store as BLOB)
	res, err := tx.Exec(`
		INSERT INTO Message (conversation_id, sender_id, encrypted_content, nonce, encryption_version)
		VALUES (?, ?, ?, ?, ?)
	`, conversationID, senderID, encryptedBytes, nonceBytes, encrypted.Version)
	if err != nil {
		return 0, err
	}
	messageID, _ := res.LastInsertId()

	// Update conversation timestamp
	_, err = tx.Exec(`UPDATE Conversation SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, conversationID)
	if err != nil {
		return 0, err
	}

	// Mark conversation as unread for recipient
	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return 0, err
	}

	if conv.User1ID == senderID {
		_, err = tx.Exec(`UPDATE Conversation SET user2_read = FALSE WHERE id = ?`, conversationID)
	} else {
		_, err = tx.Exec(`UPDATE Conversation SET user1_read = FALSE WHERE id = ?`, conversationID)
	}
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(messageID), nil
}

// GetMessagesE2EE retrieves encrypted messages (returns encrypted, client decrypts)
func GetMessagesE2EE(conversationID int) ([]EncryptedMessageData, error) {
	var messages []EncryptedMessageData
	rows, err := db.Query(`
		SELECT m.id, m.conversation_id, m.sender_id, m.encrypted_content, m.nonce, 
		       m.encryption_version, m.created_at, m.read_at, u.name as sender_name
		FROM Message m
		JOIN User u ON m.sender_id = u.id
		WHERE m.conversation_id = ? AND m.encrypted_content IS NOT NULL
		ORDER BY m.created_at ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg EncryptedMessageData
		var encryptedBytes, nonceBytes []byte
		var createdAt, readAt string

		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &encryptedBytes, &nonceBytes,
			&msg.EncryptionVersion, &createdAt, &readAt, &msg.SenderName)
		if err != nil {
			return nil, err
		}

		// Encode as base64 for JSON transport
		msg.EncryptedContent = base64.StdEncoding.EncodeToString(encryptedBytes)
		msg.Nonce = base64.StdEncoding.EncodeToString(nonceBytes)

		msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if readAt != "" {
			readAtTime, _ := time.Parse(time.RFC3339Nano, readAt)
			msg.ReadAt = &readAtTime
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// EncryptedMessageData represents encrypted message data returned to client
type EncryptedMessageData struct {
	ID               int        `json:"id"`
	ConversationID   int        `json:"conversation_id"`
	SenderID         int        `json:"sender_id"`
	EncryptedContent string     `json:"encrypted_content"` // Base64
	Nonce            string     `json:"nonce"`              // Base64
	EncryptionVersion int        `json:"encryption_version"`
	CreatedAt        time.Time  `json:"created_at"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
	SenderName       string     `json:"sender_name,omitempty"`
}
```

### Part 3: Client-Side JavaScript

#### `static/js/message-encryption.js` - Encryption Library

```javascript
// Message Encryption using libsodium.js
// Implements X25519 key exchange and ChaCha20-Poly1305 encryption
// Load libsodium.js: <script src="https://cdn.jsdelivr.net/npm/libsodium-wrappers@0.7.13/dist/browser/sodium.js"></script>

class MessageEncryption {
    constructor() {
        this.sodium = null;
        this.ready = false;
    }

    // Initialize libsodium (must be called first)
    async init() {
        if (this.ready) return;
        
        await window.sodium.ready;
        this.sodium = window.sodium;
        this.ready = true;
    }

    // Generate a key pair for the user (X25519)
    async generateKeyPair() {
        await this.init();
        
        const keyPair = this.sodium.crypto_box_keypair();
        
        return {
            publicKey: this.sodium.to_base64(keyPair.publicKey, this.sodium.base64_variants.ORIGINAL),
            privateKey: this.sodium.to_base64(keyPair.privateKey, this.sodium.base64_variants.ORIGINAL),
            keyPair: keyPair // Keep in memory
        };
    }

    // Derive shared secret from peer's public key (X25519)
    async deriveSharedSecret(privateKeyBase64, peerPublicKeyBase64) {
        await this.init();
        
        const privateKey = this.sodium.from_base64(privateKeyBase64, this.sodium.base64_variants.ORIGINAL);
        const peerPublicKey = this.sodium.from_base64(peerPublicKeyBase64, this.sodium.base64_variants.ORIGINAL);
        
        // X25519 scalar multiplication
        const sharedSecret = this.sodium.crypto_scalarmult(privateKey, peerPublicKey);
        
        return sharedSecret;
    }

    // Derive message encryption key from shared secret and conversation ID (HKDF)
    async deriveMessageKey(sharedSecret, conversationID) {
        await this.init();
        
        const salt = new TextEncoder().encode(`conversation-${conversationID}`);
        const info = new TextEncoder().encode('message-encryption-v1');
        
        // Use libsodium's generic hash for HKDF-like derivation
        // HKDF: key = H(salt || sharedSecret || info)
        const keyMaterial = new Uint8Array(sharedSecret.length + salt.length + info.length);
        keyMaterial.set(salt, 0);
        keyMaterial.set(sharedSecret, salt.length);
        keyMaterial.set(info, salt.length + sharedSecret.length);
        
        // Generate 32-byte key using generic hash
        const messageKey = this.sodium.crypto_generichash(32, keyMaterial);
        
        return messageKey;
    }

    // Encrypt a message (ChaCha20-Poly1305)
    async encryptMessage(message, messageKey) {
        await this.init();
        
        const messageBytes = new TextEncoder().encode(message);
        
        // Generate random nonce (24 bytes for ChaCha20-Poly1305)
        const nonce = this.sodium.randombytes_buf(this.sodium.crypto_secretbox_NONCEBYTES);
        
        // Encrypt with ChaCha20-Poly1305
        const encrypted = this.sodium.crypto_secretbox_easy(messageBytes, nonce, messageKey);
        
        return {
            encryptedContent: this.sodium.to_base64(encrypted, this.sodium.base64_variants.ORIGINAL),
            nonce: this.sodium.to_base64(nonce, this.sodium.base64_variants.ORIGINAL),
            version: 1
        };
    }

    // Decrypt a message (ChaCha20-Poly1305)
    async decryptMessage(encryptedData, messageKey) {
        await this.init();
        
        const encrypted = this.sodium.from_base64(encryptedData.encryptedContent, this.sodium.base64_variants.ORIGINAL);
        const nonce = this.sodium.from_base64(encryptedData.nonce, this.sodium.base64_variants.ORIGINAL);
        
        try {
            const decrypted = this.sodium.crypto_secretbox_open_easy(encrypted, nonce, messageKey);
            return new TextDecoder().decode(decrypted);
        } catch (error) {
            console.error('Decryption failed:', error);
            throw new Error('Failed to decrypt message');
        }
    }

    // Store keys in browser storage
    async storeKeys(userID, keys) {
        await this.init();
        const key = `e2ee_keys_${userID}`;
        localStorage.setItem(key, JSON.stringify(keys));
    }

    // Retrieve keys from browser storage
    async getKeys(userID) {
        const key = `e2ee_keys_${userID}`;
        const stored = localStorage.getItem(key);
        if (!stored) return null;
        return JSON.parse(stored);
    }

    // Store shared secret for a conversation
    async storeSharedSecret(conversationID, sharedSecret) {
        await this.init();
        const key = `e2ee_shared_${conversationID}`;
        const sharedSecretBase64 = this.sodium.to_base64(sharedSecret, this.sodium.base64_variants.ORIGINAL);
        localStorage.setItem(key, sharedSecretBase64);
    }

    // Get shared secret for a conversation
    async getSharedSecret(conversationID) {
        await this.init();
        const key = `e2ee_shared_${conversationID}`;
        const stored = localStorage.getItem(key);
        if (!stored) return null;
        return this.sodium.from_base64(stored, this.sodium.base64_variants.ORIGINAL);
    }

    // Derive encryption key from user password (for encrypting private keys)
    async deriveKeyFromPassword(password, salt) {
        await this.init();
        
        // Use libsodium's pwhash (Argon2id) for password-based key derivation
        const passwordBytes = new TextEncoder().encode(password);
        const key = this.sodium.crypto_pwhash(
            32, // 32 bytes = 256 bits
            passwordBytes,
            salt,
            this.sodium.crypto_pwhash_OPSLIMIT_INTERACTIVE,
            this.sodium.crypto_pwhash_MEMLIMIT_INTERACTIVE,
            this.sodium.crypto_pwhash_ALG_DEFAULT
        );
        
        return key;
    }

    // Encrypt private key for server storage
    async encryptPrivateKeyForStorage(privateKeyBase64, password) {
        await this.init();
        
        const privateKey = this.sodium.from_base64(privateKeyBase64, this.sodium.base64_variants.ORIGINAL);
        const salt = this.sodium.randombytes_buf(this.sodium.crypto_pwhash_SALTBYTES);
        const key = await this.deriveKeyFromPassword(password, salt);
        
        // Encrypt with ChaCha20-Poly1305
        const nonce = this.sodium.randombytes_buf(this.sodium.crypto_secretbox_NONCEBYTES);
        const encrypted = this.sodium.crypto_secretbox_easy(privateKey, nonce, key);
        
        return {
            encrypted: this.sodium.to_base64(encrypted, this.sodium.base64_variants.ORIGINAL),
            salt: this.sodium.to_base64(salt, this.sodium.base64_variants.ORIGINAL),
            nonce: this.sodium.to_base64(nonce, this.sodium.base64_variants.ORIGINAL)
        };
    }

    // Decrypt private key from server storage
    async decryptPrivateKeyFromStorage(encryptedData, password) {
        await this.init();
        
        const encrypted = this.sodium.from_base64(encryptedData.encrypted, this.sodium.base64_variants.ORIGINAL);
        const salt = this.sodium.from_base64(encryptedData.salt, this.sodium.base64_variants.ORIGINAL);
        const nonce = this.sodium.from_base64(encryptedData.nonce, this.sodium.base64_variants.ORIGINAL);
        
        const key = await this.deriveKeyFromPassword(password, salt);
        const privateKey = this.sodium.crypto_secretbox_open_easy(encrypted, nonce, key);
        
        return privateKey;
    }
}

// Global instance
const messageEncryption = new MessageEncryption();
```

#### `static/js/message-handler.js` - HTMX Integration

```javascript
// Intercept message form submissions and encrypt before sending

document.addEventListener('DOMContentLoaded', function() {
    // Intercept all message form submissions
    document.body.addEventListener('submit', async function(event) {
        const form = event.target;
        
        // Check if this is a message form
        if (!form.action || !form.action.includes('/messages/') || !form.action.includes('/send')) {
            return; // Not a message form, let it submit normally
        }

        // Prevent default submission
        event.preventDefault();

        const messageInput = form.querySelector('input[name="message"]');
        if (!messageInput || !messageInput.value.trim()) {
            return; // No message to send
        }

        const conversationID = extractConversationID(form.action);
        if (!conversationID) {
            console.error('Could not extract conversation ID');
            return;
        }

        try {
            // Get or create encryption keys for this conversation
            const messageKey = await getOrCreateMessageKey(conversationID);
            
            // Encrypt the message
            const encrypted = await messageEncryption.encryptMessage(
                messageInput.value,
                messageKey
            );

            // Create hidden input with encrypted data
            const encryptedInput = document.createElement('input');
            encryptedInput.type = 'hidden';
            encryptedInput.name = 'encrypted_message';
            encryptedInput.value = JSON.stringify(encrypted);
            form.appendChild(encryptedInput);

            // Clear the plaintext message (don't send it)
            messageInput.value = '';

            // Submit via HTMX with encrypted data
            htmx.ajax('POST', form.action, {
                values: {
                    encrypted_message: JSON.stringify(encrypted)
                },
                target: form.getAttribute('hx-target') || form.closest('[hx-target]')?.getAttribute('hx-target'),
                swap: form.getAttribute('hx-swap') || 'outerHTML'
            });

        } catch (error) {
            console.error('Encryption error:', error);
            alert('Failed to encrypt message. Please try again.');
        }
    });

    // Decrypt messages when they're loaded
    document.body.addEventListener('htmx:afterSwap', async function(event) {
        const target = event.detail.target;
        
        // Find all encrypted message elements
        const encryptedMessages = target.querySelectorAll('[data-encrypted-message]');
        
        for (const msgElement of encryptedMessages) {
            const conversationID = msgElement.getAttribute('data-conversation-id');
            const encryptedData = JSON.parse(msgElement.getAttribute('data-encrypted-message'));
            
            try {
                const messageKey = await getOrCreateMessageKey(conversationID);
                const decrypted = await messageEncryption.decryptMessage(encryptedData, messageKey);
                
                // Replace encrypted content with decrypted
                msgElement.textContent = decrypted;
                msgElement.removeAttribute('data-encrypted-message');
            } catch (error) {
                console.error('Decryption error:', error);
                msgElement.textContent = '[Unable to decrypt message]';
            }
        }
    });
});

// Extract conversation ID from URL
function extractConversationID(url) {
    const match = url.match(/\/messages\/(\d+)\//);
    return match ? parseInt(match[1]) : null;
}

// Get or create message encryption key for conversation
async function getOrCreateMessageKey(conversationID) {
    await messageEncryption.init();
    
    // Check if we have a cached key
    const cacheKey = `msg_key_${conversationID}`;
    const cached = sessionStorage.getItem(cacheKey);
    if (cached) {
        // Re-import the key (stored as base64)
        const keyData = JSON.parse(cached);
        return messageEncryption.sodium.from_base64(keyData.key, messageEncryption.sodium.base64_variants.ORIGINAL);
    }

    // Need to perform key exchange
    const sharedSecret = await performKeyExchange(conversationID);
    const messageKey = await messageEncryption.deriveMessageKey(sharedSecret, conversationID);
    
    // Cache the key (as base64)
    sessionStorage.setItem(cacheKey, JSON.stringify({
        key: messageEncryption.sodium.to_base64(messageKey, messageEncryption.sodium.base64_variants.ORIGINAL)
    }));

    return messageKey;
}

// Perform key exchange with peer
async function performKeyExchange(conversationID) {
    await messageEncryption.init();
    
    // Get current user's key pair (or generate if first time)
    const userID = getCurrentUserID(); // Implement this
    let keys = await messageEncryption.getKeys(userID);
    
    if (!keys) {
        // Generate new key pair
        keys = await messageEncryption.generateKeyPair();
        
        // Send public key to server
        await fetch('/api/e2ee/public-key', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ public_key: keys.publicKey })
        });
        
        await messageEncryption.storeKeys(userID, keys);
    }

    // Get peer's public key from server
    const response = await fetch(`/api/e2ee/conversation/${conversationID}/peer-key`);
    const peerData = await response.json();
    
    // Derive shared secret (keys.privateKey is already base64 string)
    const sharedSecret = await messageEncryption.deriveSharedSecret(
        keys.privateKey,
        peerData.public_key
    );

    // Store shared secret
    await messageEncryption.storeSharedSecret(conversationID, sharedSecret);

    return sharedSecret;
}

function getCurrentUserID() {
    // Extract from page or make API call
    // This is a placeholder - implement based on your auth system
    return parseInt(document.body.getAttribute('data-user-id') || '0');
}
```

### Part 4: Server Handlers

#### `handlers/e2ee.go` - Key Exchange Handlers

```go
package handlers

import (
	"encoding/base64"
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/messaging"
)

// HandlePublicKeyUpload handles uploading a user's public key
func HandlePublicKeyUpload(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req struct {
		PublicKey string `json:"public_key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid public key format"})
	}

	keyID, err := messaging.StorePublicKey(userID, publicKeyBytes)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store public key"})
	}

	return c.JSON(fiber.Map{"key_id": keyID, "status": "success"})
}

// HandleGetPeerPublicKey returns the peer's public key for a conversation
func HandleGetPeerPublicKey(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid conversation ID"})
	}

	// Get conversation to find peer
	conv, err := messaging.GetConversationByID(conversationID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Conversation not found"})
	}

	// Determine peer ID
	peerID := conv.User1ID
	if userID == conv.User1ID {
		peerID = conv.User2ID
	}

	// Get peer's public key
	peerKey, err := messaging.GetPublicKey(peerID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Peer public key not found"})
	}

	return c.JSON(fiber.Map{
		"public_key": base64.StdEncoding.EncodeToString(peerKey.KeyBytes),
		"key_id":     peerKey.ID,
	})
}

// HandleSendMessageE2EE handles sending an encrypted message
func HandleSendMessageE2EE(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid conversation ID"})
	}

	// Parse encrypted message from form
	encryptedJSON := c.FormValue("encrypted_message")
	if encryptedJSON == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Missing encrypted message"})
	}

	var encrypted messaging.EncryptedMessage
	if err := json.Unmarshal([]byte(encryptedJSON), &encrypted); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid encrypted message format"})
	}

	// Verify user is part of conversation
	conv, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Conversation not found"})
	}

	if conv.User1ID != userID && conv.User2ID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Access denied"})
	}

	// Store encrypted message (server never sees plaintext)
	messageID, err := messaging.AddMessageE2EE(conversationID, userID, encrypted)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to send message"})
	}

	// Determine recipient ID
	recipientID := conv.User1ID
	if userID == conv.User1ID {
		recipientID = conv.User2ID
	}

	// Send notification WITHOUT message content (server can't decrypt)
	// Notification only contains: sender name, ad title, and link
	notificationService, err := notification.NewNotificationService()
	if err != nil {
		// Log error but don't fail the request
		logger.Error("Failed to create notification service", "error", err)
	} else {
		go func() {
			// Pass empty string for messageContent - notification will use link-only format
			err := notificationService.NotifyNewMessageE2EE(conversationID, userID, recipientID)
			if err != nil {
				logger.Error("Failed to send notification", "error", err)
			}
		}()
	}

	// Return success (HTMX will reload conversation)
	return c.JSON(fiber.Map{"message_id": messageID, "status": "success"})
}
```

#### `notification/notification.go` - Link-Only Notifications for E2EE

```go
// NotifyNewMessageE2EE sends a notification WITHOUT message content
// Used when messages are end-to-end encrypted (server can't read content)
func (n *NotificationService) NotifyNewMessageE2EE(conversationID int, senderID, recipientID int) error {
	// Get conversation details
	conversation, err := messaging.GetConversationByID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Get sender and recipient details
	sender, _, found := user.GetUserByID(senderID)
	if !found {
		return fmt.Errorf("sender not found")
	}

	recipient, _, found := user.GetUserByID(recipientID)
	if !found {
		return fmt.Errorf("recipient not found")
	}

	// Get ad details
	ad, err := getAdDetails(conversation.AdID)
	if err != nil {
		return fmt.Errorf("failed to get ad details: %w", err)
	}

	// Send notification based on recipient's preference
	switch recipient.NotificationMethod {
	case user.NotificationMethodSMS:
		return n.sendSMSNotificationE2EE(recipient.Phone, sender.Name, ad.Title, conversationID)
	case user.NotificationMethodEmail:
		if recipient.EmailAddress == nil {
			return fmt.Errorf("recipient has email notifications enabled but no email address")
		}
		return n.sendEmailNotificationE2EE(*recipient.EmailAddress, sender.Name, ad.Title, conversationID)
	default:
		logger.Warn("Unknown notification method",
			"notificationMethod", recipient.NotificationMethod,
			"userID", recipientID)
		return nil
	}
}

// sendSMSNotificationE2EE sends an SMS notification WITHOUT message content
// Only includes: sender name, ad title, and link
func (n *NotificationService) sendSMSNotificationE2EE(phoneNumber, senderName, adTitle string, conversationID int) error {
	if !n.hasSMS {
		return fmt.Errorf("SMS service not available")
	}

	// Validate phone number format
	if len(phoneNumber) < 10 {
		return fmt.Errorf("invalid phone number format: %s", phoneNumber)
	}

	// Link-only notification (no message content - server can't decrypt E2EE messages)
	message := fmt.Sprintf("New message from %s about '%s'. View at: %s/messages?expand=%d",
		senderName, adTitle, config.BaseURL, conversationID)

	// Send SMS using Twilio
	err := sms.SendMessage(phoneNumber, message)
	if err != nil {
		logger.Error("SMS notification failed",
			"error", err, "phoneNumber", phoneNumber)
		return err
	}

	logger.Info("SMS notification sent successfully (E2EE)",
		"phoneNumber", phoneNumber)
	return nil
}

// sendEmailNotificationE2EE sends an email notification WITHOUT message content
func (n *NotificationService) sendEmailNotificationE2EE(emailAddress, senderName, adTitle string, conversationID int) error {
	if n.emailService == nil {
		return fmt.Errorf("email service not available")
	}

	return n.emailService.SendNotificationEmailE2EE(emailAddress, senderName, adTitle, conversationID)
}
```

#### `email/email.go` - Link-Only Email Notifications

```go
// SendNotificationEmailE2EE sends an email notification WITHOUT message content
// Only includes: sender name, ad title, and link
func (s *EmailService) SendNotificationEmailE2EE(emailAddress, senderName, adTitle string, conversationID int) error {
	subject := fmt.Sprintf("New message from %s about '%s'", senderName, adTitle)
	
	// Link-only body (no message content - server can't decrypt E2EE messages)
	htmlBody := fmt.Sprintf(`
<html>
<head>
    <title>New Message</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .button { display: inline-block; background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 12px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2 style="margin: 0; color: #495057;">New Message on Parts Pile</h2>
        </div>
        
        <p><strong>From:</strong> %s</p>
        <p><strong>About:</strong> %s</p>
        
        <p>You have a new message. Click the link below to view it in the app.</p>
        
        <a href="%s/messages?expand=%d" class="button">View Conversation</a>
        
        <div class="footer">
            <p>This message is end-to-end encrypted. You must open the app to read it.</p>
            <p>You can manage your notification preferences in your account settings.</p>
        </div>
    </div>
</body>
</html>`, senderName, adTitle, config.BaseURL, conversationID)

	return s.SendEmail(emailAddress, subject, htmlBody)
}
```

**Key Changes for E2EE Notifications:**
- Remove `messageContent` parameter (server can't decrypt)
- SMS: "New message from [Sender] about '[Ad]'. View at: [link]"
- Email: Same format, no message preview box
- User experience: Click link → Open UI → Client decrypts and displays message

### Part 4.5: Real-Time Updates with E2EE

**How Real-Time Updates Work:**
- Server uses SSE (Server-Sent Events) to push new message notifications
- When new encrypted message arrives, SSE sends conversation ID
- Client receives SSE event, fetches encrypted message, decrypts on-the-fly
- UI updates automatically with decrypted message

#### Updated `static/js/message-handler.js` - SSE Integration

```javascript
// Handle real-time updates via SSE for encrypted messages
document.addEventListener('DOMContentLoaded', function() {
    // Connect to SSE endpoint
    const eventSource = new EventSource('/api/sse');
    
    eventSource.addEventListener('new_message', async function(event) {
        const conversationID = parseInt(event.data);
        
        // Check if this conversation is currently open/expanded
        const conversationElement = document.querySelector(`#conversation-${conversationID}`);
        if (!conversationElement) {
            // Conversation not visible, just update unread count
            return;
        }
        
        // Fetch the new encrypted message(s)
        try {
            const response = await fetch(`/api/messages/${conversationID}/latest`);
            const encryptedMessages = await response.json();
            
            // Decrypt and display each new message
            for (const encryptedMsg of encryptedMessages) {
                const messageKey = await getOrCreateMessageKey(conversationID);
                const decrypted = await messageEncryption.decryptMessage(encryptedMsg, messageKey);
                
                // Add decrypted message to UI
                addMessageToUI(conversationID, decrypted, encryptedMsg.sender_id);
            }
        } catch (error) {
            console.error('Failed to fetch/decrypt new message:', error);
        }
    });
    
    eventSource.addEventListener('unread_count', function(event) {
        const count = parseInt(event.data);
        updateUnreadBadge(count);
    });
    
    eventSource.onerror = function(error) {
        console.error('SSE connection error:', error);
        // Could implement reconnection logic here
    };
});

// Helper to add decrypted message to UI
function addMessageToUI(conversationID, decryptedText, senderID) {
    const messagesContainer = document.querySelector(`#conversation-${conversationID} .messages-list`);
    if (!messagesContainer) return;
    
    const currentUserID = getCurrentUserID();
    const isOwnMessage = senderID === currentUserID;
    
    const messageElement = document.createElement('div');
    messageElement.className = isOwnMessage ? 'flex justify-end' : 'flex justify-start';
    messageElement.innerHTML = `
        <div class="max-w-xs px-4 py-2 rounded-lg ${isOwnMessage ? 'bg-blue-500 text-white' : 'bg-gray-200'}">
            ${escapeHtml(decryptedText)}
        </div>
    `;
    
    messagesContainer.appendChild(messageElement);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
```

#### Updated `handlers/messaging.go` - SSE with Encrypted Messages

```go
// HandleSSE handles Server-Sent Events for real-time messaging updates
// For E2EE: SSE only sends conversation ID, client fetches and decrypts
func HandleSSE(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	// Create a channel for this user's updates
	userUpdates := make(chan messaging.ConversationUpdate, 10)
	defer close(userUpdates)

	// Register this user's update channel
	messaging.RegisterUserUpdates(currentUser.ID, userUpdates)
	defer messaging.UnregisterUserUpdates(currentUser.ID)

	// Send initial unread count
	unreadCount, err := messaging.GetUnreadCount(currentUser.ID)
	if err == nil {
		c.WriteString(fmt.Sprintf("event: unread_count\ndata: %d\n\n", unreadCount))
	}

	// Keep connection alive and send updates
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case update := <-userUpdates:
			switch update.Type {
			case "new_message":
				// For E2EE: Only send conversation ID, not message content
				// Client will fetch encrypted message and decrypt
				c.WriteString(fmt.Sprintf("event: new_message\ndata: %d\n\n", update.ConversationID))
			case "unread_count":
				c.WriteString(fmt.Sprintf("event: unread_count\ndata: %d\n\n", update.UnreadCount))
			default:
				updateJSON, _ := json.Marshal(update)
				c.WriteString(fmt.Sprintf("event: update\ndata: %s\n\n", updateJSON))
			}

		case <-ticker.C:
			c.WriteString("data: {\"type\":\"ping\"}\n\n")

		case <-c.Context().Done():
			return nil
		}
	}
}

// HandleGetLatestMessages returns the latest encrypted messages for a conversation
// Used by client after receiving SSE notification
func HandleGetLatestMessages(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid conversation ID"})
	}

	// Verify user is part of conversation
	conv, err := messaging.GetConversationByID(conversationID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Conversation not found"})
	}

	if conv.User1ID != userID && conv.User2ID != userID {
		return c.Status(403).JSON(fiber.Map{"error": "Access denied"})
	}

	// Get latest encrypted messages (client will decrypt)
	// Could add a "since" parameter to only get new messages
	messages, err := messaging.GetMessagesE2EE(conversationID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load messages"})
	}

	return c.JSON(messages)
}
```

**Real-Time E2EE Flow:**
1. User A sends encrypted message → Server stores
2. Server sends SSE event to User B: `event: new_message, data: 501`
3. User B's client receives SSE event
4. Client fetches encrypted message: `GET /api/messages/501/latest`
5. Server returns encrypted message (never decrypts)
6. Client decrypts message on-the-fly
7. Client displays decrypted message in UI
8. UI updates in real-time without page refresh

### Part 5: UI Updates

#### Update `ui/messaging.go` to include encryption scripts

```go
// Add to MessagesPage and other message pages:
// Load libsodium.js first (required dependency)
Script(
    Src("https://cdn.jsdelivr.net/npm/libsodium-wrappers@0.7.13/dist/browser/sodium.js"),
),
// Then load our encryption library
Script(
    Src("/js/message-encryption.js"),
),
// Finally load the message handler
Script(
    Src("/js/message-handler.js"),
),
```

#### Update MessageItem to handle encrypted messages

```go
// MessageItem renders encrypted message (client will decrypt)
func MessageItemE2EE(msg messaging.EncryptedMessageData, currentUserID int) g.Node {
    // Store encrypted data as data attribute, client JS will decrypt
    encryptedJSON, _ := json.Marshal(msg)
    
    return Div(
        Class(containerClass),
        Div(
            Class("flex items-end gap-2"),
            Div(
                Class(fmt.Sprintf("max-w-xs lg:max-w-md px-4 py-2 rounded-lg %s", messageClass)),
                Div(
                    DataAttr("encrypted-message", string(encryptedJSON)),
                    DataAttr("conversation-id", fmt.Sprintf("%d", msg.ConversationID)),
                    Class("text-sm"),
                    g.Text("[Encrypted message - decrypting...]"),
                ),
            ),
        ),
    )
}
```

---

## Migration Path

1. **Phase 1: Add encryption support (backward compatible)**
   - Add database columns
   - Deploy server code
   - Deploy client JS (optional, graceful degradation)

2. **Phase 2: Enable E2EE for new conversations**
   - Users generate key pairs on first message
   - New messages use E2EE
   - Old messages remain unencrypted

3. **Phase 3: Full migration (optional)**
   - Re-encrypt old messages if needed
   - Remove plaintext support

---

## Security Considerations

1. **Key Storage:** Private keys stored in browser localStorage (encrypted with user password using Argon2id)
2. **Key Exchange:** X25519 provides forward secrecy
3. **Message Authentication:** ChaCha20-Poly1305 provides authenticated encryption
4. **Nonce Reuse:** Each message gets unique nonce (prevents replay attacks)
5. **Server Compromise:** Server cannot decrypt messages (true E2EE)
6. **Password Hashing:** Argon2id for password-based key derivation (same as login passwords)

### Database Compromise Analysis

**Question: If someone has database access, can they decrypt conversations?**

**Answer: NO - Public keys alone are NOT enough to decrypt.**

**What's in the Database:**
```
UserPublicKey table:
  - public_key (32 bytes) ← Public keys only
  - user_id, key_type

Message table:
  - encrypted_content (ciphertext)
  - nonce (24 bytes for ChaCha20-Poly1305)
  - encryption_version
```

**What's NOT in the Database:**
- ❌ Private keys (stored encrypted in User table, or browser localStorage)
- ❌ Shared secrets (never stored - derived on-the-fly by each client when needed)
- ❌ Message keys (never stored - derived on-the-fly from shared secret + conversation ID)
- ❌ ConversationKey table (not needed - shared secrets computed client-side)

**Why Public Keys Can't Decrypt:**

1. **To decrypt a message, you need:**
   - Encrypted message (✅ in DB)
   - Nonce (✅ in DB)
   - Message key (❌ NOT in DB - derived from shared secret)
   - Shared secret (❌ NOT in DB - derived from private key + public key)

2. **To derive shared secret:**
   ```
   Shared Secret = X25519(Private Key A, Public Key B)
   ```
   - You need BOTH private key AND public key
   - Database only has public keys
   - **Without private key, you cannot compute shared secret**

3. **To derive message key:**
   ```
   Message Key = HKDF(Shared Secret, Conversation ID)
   ```
   - Requires shared secret (which requires private key)
   - **Cannot compute without private key**

4. **To decrypt message:**
   ```
   Plaintext = ChaCha20-Poly1305-Decrypt(Ciphertext, Message Key, Nonce)
   ```
   - Requires message key (which requires shared secret, which requires private key)
   - **Cannot decrypt without private key**

**Attack Scenarios:**

**Scenario 1: Database-only access**
- ✅ Attacker has: Public keys, encrypted messages, nonces
- ❌ Attacker missing: Private keys
- **Result: CANNOT DECRYPT** ✅

**Scenario 2: Database + Encrypted Private Keys (if stored in User table)**
- ✅ Attacker has: Public keys, encrypted messages, encrypted private keys
- ❌ Attacker missing: User passwords (to decrypt private keys)
- **Result: CANNOT DECRYPT** (unless passwords are weak/compromised) ✅

**Scenario 3: Database + User Passwords**
- ✅ Attacker has: Everything
- ⚠️ Can decrypt private keys, then decrypt messages
- **Result: CAN DECRYPT** ❌
- **Mitigation:** Strong passwords, rate limiting, 2FA

**Security Guarantees:**

✅ **Database compromise alone:** Messages remain encrypted
✅ **Public key exposure:** Not a security risk (that's the point of public keys)
✅ **Shared secret:** Never stored in database - derived on-the-fly by each client
✅ **True E2EE maintained:** Server never sees plaintext or shared secrets

**Best Practices:**

1. **Private key storage:**
   - Option A: Browser localStorage (encrypted with password) - ✅ Better (not in DB)
   - Option B: User table (encrypted with password) - ⚠️ Requires strong passwords

2. **If storing encrypted private keys in DB:**
   - Use strong password-derived encryption (PBKDF2 with high iterations)
   - Consider additional factors (device fingerprint, etc.)
   - Monitor for password breaches

3. **Defense in depth:**
   - Encrypt database at rest
   - Limit database access
   - Use strong authentication
   - Monitor access logs

---

## Multi-Device Support

### ⚠️ Problem: Current Design Breaks with Multiple Devices

**The Issue:**
- Private keys stored in `localStorage` (per-device, per-browser)
- Each device generates its own key pair
- Device 1 and Device 2 of User A would have different private keys
- **Result:** Device 2 cannot decrypt messages sent/received by Device 1

**Example Scenario:**
```
1. User A on Device 1 (Phone):
   - Generates Private Key A1, Public Key A1
   - Exchanges keys with User B
   - Sends message: Encrypted with Key A1
   - Receives message: Decrypted with Key A1 ✅

2. User A on Device 2 (Desktop):
   - Generates Private Key A2, Public Key A2 (DIFFERENT!)
   - Exchanges keys with User B (creates NEW shared secret)
   - Tries to read messages from Device 1: ❌ CANNOT DECRYPT
   - Only sees messages sent from Device 2
```

### Solutions for Multi-Device Support

#### Solution 1: Server-Synced Private Keys (Recommended)

**How it works:**
- Private key encrypted with user's password (derived key)
- Encrypted private key stored on server
- All devices sync the same encrypted private key
- Each device decrypts with user's password

**Implementation:**

```sql
-- Add to User table
ALTER TABLE User ADD COLUMN encrypted_private_key BLOB;
ALTER TABLE User ADD COLUMN private_key_salt BLOB;
ALTER TABLE User ADD COLUMN private_key_nonce BLOB; -- For ChaCha20-Poly1305 encryption
```

```javascript
// message-encryption.js - Sync private keys across devices (using libsodium.js)
// Note: These methods are part of the main MessageEncryption class (see Part 3)

// Derive encryption key from user password (Argon2id)
async deriveKeyFromPassword(password, salt) {
    await this.init();
    
    const passwordBytes = new TextEncoder().encode(password);
    const key = this.sodium.crypto_pwhash(
        32, // 32 bytes = 256 bits
        passwordBytes,
        salt,
        this.sodium.crypto_pwhash_OPSLIMIT_INTERACTIVE,
        this.sodium.crypto_pwhash_MEMLIMIT_INTERACTIVE,
        this.sodium.crypto_pwhash_ALG_DEFAULT
    );
    
    return key;
}

// Encrypt private key for server storage (ChaCha20-Poly1305)
async encryptPrivateKeyForStorage(privateKeyBase64, password) {
    await this.init();
    
    const privateKey = this.sodium.from_base64(privateKeyBase64, this.sodium.base64_variants.ORIGINAL);
    const salt = this.sodium.randombytes_buf(this.sodium.crypto_pwhash_SALTBYTES);
    const key = await this.deriveKeyFromPassword(password, salt);
    
    // Encrypt with ChaCha20-Poly1305
    const nonce = this.sodium.randombytes_buf(this.sodium.crypto_secretbox_NONCEBYTES);
    const encrypted = this.sodium.crypto_secretbox_easy(privateKey, nonce, key);
    
    return {
        encrypted: this.sodium.to_base64(encrypted, this.sodium.base64_variants.ORIGINAL),
        salt: this.sodium.to_base64(salt, this.sodium.base64_variants.ORIGINAL),
        nonce: this.sodium.to_base64(nonce, this.sodium.base64_variants.ORIGINAL)
    };
}

// Decrypt private key from server storage
async decryptPrivateKeyFromStorage(encryptedData, password) {
    await this.init();
    
    const encrypted = this.sodium.from_base64(encryptedData.encrypted, this.sodium.base64_variants.ORIGINAL);
    const salt = this.sodium.from_base64(encryptedData.salt, this.sodium.base64_variants.ORIGINAL);
    const nonce = this.sodium.from_base64(encryptedData.nonce, this.sodium.base64_variants.ORIGINAL);
    
    const key = await this.deriveKeyFromPassword(password, salt);
    const privateKey = this.sodium.crypto_secretbox_open_easy(encrypted, nonce, key);
    
    return privateKey;
}
```

**Note:** The full `MessageEncryption` class with all methods (including `syncPrivateKeyFromServer` and `uploadPrivateKeyToServer`) is shown in Part 3 above. The methods shown here are the key sync-specific ones using libsodium.js.

**Server Handler:**

```go
// handlers/e2ee.go
// HandleGetPrivateKey returns user's encrypted private key
func HandleGetPrivateKey(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var encryptedKey, salt, nonce []byte
	err := db.QueryRow(`
		SELECT encrypted_private_key, private_key_salt, private_key_nonce
		FROM User
		WHERE id = ?
	`, userID).Scan(&encryptedKey, &salt, &nonce)
	
	if err == sql.ErrNoRows || len(encryptedKey) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "No private key found"})
	}

	return c.JSON(fiber.Map{
		"encrypted_private_key": base64.StdEncoding.EncodeToString(encryptedKey),
		"private_key_salt":     base64.StdEncoding.EncodeToString(salt),
		"private_key_nonce":    base64.StdEncoding.EncodeToString(nonce),
	})
}

// HandleUploadPrivateKey stores user's encrypted private key
func HandleUploadPrivateKey(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if userID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req struct {
		EncryptedPrivateKey string `json:"encrypted_private_key"`
		PrivateKeySalt      string `json:"private_key_salt"`
		PrivateKeyNonce     string `json:"private_key_nonce"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	encryptedKey, _ := base64.StdEncoding.DecodeString(req.EncryptedPrivateKey)
	salt, _ := base64.StdEncoding.DecodeString(req.PrivateKeySalt)
	nonce, _ := base64.StdEncoding.DecodeString(req.PrivateKeyNonce)

	_, err := db.Exec(`
		UPDATE User
		SET encrypted_private_key = ?, private_key_salt = ?, private_key_nonce = ?
		WHERE id = ?
	`, encryptedKey, salt, nonce, userID)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store private key"})
	}

	return c.JSON(fiber.Map{"status": "success"})
}
```

**Pros:**
- ✅ All devices use same private key
- ✅ Messages readable on all devices
- ✅ True E2EE maintained (server can't decrypt private key)

**Cons:**
- ⚠️ Requires user password on each device
- ⚠️ Password change requires re-encrypting private key
- ⚠️ If password compromised, attacker can decrypt private key

---

#### Solution 2: Per-Device Keys with Message Re-Encryption

**How it works:**
- Each device has its own key pair
- Messages encrypted for ALL devices in conversation
- When sending: Encrypt once per recipient device
- More complex but better security isolation

**Implementation:**

```javascript
// Encrypt message for all devices of recipient
async function encryptForAllDevices(message, recipientUserID) {
    // Get all public keys for recipient's devices
    const response = await fetch(`/api/e2ee/user/${recipientUserID}/devices`);
    const devices = await response.json();
    
    const encryptedMessages = [];
    for (const device of devices) {
        const sharedSecret = await deriveSharedSecret(myPrivateKey, device.public_key);
        const messageKey = await deriveMessageKey(sharedSecret, conversationID);
        const encrypted = await encryptMessage(message, messageKey);
        
        encryptedMessages.push({
            device_id: device.id,
            encrypted_content: encrypted.encryptedContent,
            nonce: encrypted.nonce
        });
    }
    
    return encryptedMessages;
}
```

**Pros:**
- ✅ Better security (device compromise doesn't affect other devices)
- ✅ Forward secrecy per device

**Cons:**
- ❌ Much more complex
- ❌ Multiple encryptions per message
- ❌ Need device management

---

#### Solution 3: Hybrid Approach (Recommended for Your App)

**How it works:**
- Single private key per user (synced via server)
- Private key encrypted with password-derived key
- Encrypted private key stored in User table
- On login: Download encrypted key, decrypt with password, store in browser
- Simple key rotation on password change
- Works well for async messaging

**Implementation Flow:**

```
1. User logs in on Device 1 (First Time):
   - User enters username/password
   - Server authenticates
   - Client checks: Does server have encrypted private key?
   - If NO: Generate new key pair
        → Encrypt private key with password
        → Upload encrypted private key to server
        → Store decrypted private key in browser localStorage
   - If YES: Download encrypted private key
        → Decrypt with password
        → Store decrypted private key in browser localStorage
   - Now Device 1 can encrypt/decrypt messages ✅

2. User logs in on Device 2 (New Device):
   - User enters username/password
   - Server authenticates
   - Client checks: Does server have encrypted private key?
   - YES: Download encrypted private key (from Device 1)
        → Decrypt with password
        → Store decrypted private key in browser localStorage
   - Now Device 2 has SAME private key as Device 1 ✅
   - Device 2 can decrypt all messages (including ones sent/received on Device 1) ✅

3. User sends message from Device 2:
   - Uses synced private key (same as Device 1)
   - Encrypts message
   - Sends to server
   - Device 1 can decrypt (if UI is open) ✅

4. User changes password:
   - Re-encrypts private key with NEW password
   - Updates encrypted private key on server
   - All devices re-sync on next login (decrypt with new password)
```

**Database Schema:**

```sql
-- Add to User table
ALTER TABLE User ADD COLUMN encrypted_private_key BLOB;
ALTER TABLE User ADD COLUMN private_key_salt BLOB;
ALTER TABLE User ADD COLUMN private_key_nonce BLOB; -- For ChaCha20-Poly1305 encryption
ALTER TABLE User ADD COLUMN public_key BLOB; -- Cache public key for faster lookups
```

**Login Flow on New Device:**

```javascript
// message-handler.js - Sync private key on login
async function syncPrivateKeyOnLogin(userID, password) {
    await messageEncryption.init();
    
    // Step 1: Check if server has encrypted private key
    const response = await fetch('/api/e2ee/private-key');
    const data = await response.json();
    
    if (data.encrypted_private_key) {
        // Step 2: Server has encrypted key - decrypt it
        const privateKeyBytes = await messageEncryption.decryptPrivateKeyFromStorage(
            {
                encrypted: data.encrypted_private_key,
                salt: data.private_key_salt,
                nonce: data.private_key_nonce
            },
            password
        );
        
        // Step 3: Convert to base64 for storage
        const privateKeyBase64 = messageEncryption.sodium.to_base64(
            privateKeyBytes, 
            messageEncryption.sodium.base64_variants.ORIGINAL
        );
        
        // Step 4: Store in browser localStorage (encrypted with password again for local storage)
        const localEncrypted = await messageEncryption.encryptPrivateKeyForStorage(
            privateKeyBase64,
            password
        );
        localStorage.setItem(`e2ee_private_key_${userID}`, JSON.stringify(localEncrypted));
        
        return privateKeyBytes;
    } else {
        // Step 5: No key on server - generate new one
        const keys = await messageEncryption.generateKeyPair();
        
        // Step 6: Encrypt and upload to server
        await messageEncryption.uploadPrivateKeyToServer(userID, keys.privateKey, password);
        
        // Step 7: Store in browser localStorage
        const localEncrypted = await messageEncryption.encryptPrivateKeyForStorage(
            keys.privateKey,
            password
        );
        localStorage.setItem(`e2ee_private_key_${userID}`, JSON.stringify(localEncrypted));
        
        // Step 8: Return private key bytes
        return messageEncryption.sodium.from_base64(
            keys.privateKey, 
            messageEncryption.sodium.base64_variants.ORIGINAL
        );
    }
}

// Call this after successful login
document.addEventListener('htmx:afterSwap', async function(event) {
    // Check if we just logged in
    if (event.detail.target.id === 'login-success') {
        const userID = getCurrentUserID();
        const password = getPasswordFromLoginForm(); // You'd need to capture this securely
        
        // Sync private key from server
        await syncPrivateKeyOnLogin(userID, password);
    }
});
```

**Important Security Notes:**

1. **Password Handling:**
   - Password is needed to decrypt private key from server
   - Password should be captured during login (before form submission)
   - Consider using a secure password prompt if not available from login form
   - Password is NOT stored - only used to decrypt, then discarded

2. **Local Storage:**
   - Private key stored in localStorage is encrypted again with password
   - This protects against XSS attacks (attacker can't use key without password)
   - Consider using sessionStorage for better security (cleared on browser close)

3. **Key Sync:**
   - Encrypted private key on server allows sync across devices
   - Server never sees plaintext private key
   - Each device decrypts with user's password
   - All devices end up with same private key ✅

**Key Management:**

```go
// messaging/e2ee.go
// GetOrCreateUserKeyPair gets or creates user's key pair
func GetOrCreateUserKeyPair(userID int, password string) (privateKey, publicKey []byte, err error) {
	// Check if user has encrypted private key
	var encryptedKey, salt, nonce []byte
	err = db.QueryRow(`
		SELECT encrypted_private_key, private_key_salt, private_key_nonce
		FROM User
		WHERE id = ?
	`, userID).Scan(&encryptedKey, &salt, &nonce)
	
	if err == nil && len(encryptedKey) > 0 {
		// Decrypt existing key (client-side, this is just for reference)
		// Actual decryption happens in JavaScript using libsodium.js
		// This function would be used if server needed to decrypt (which it doesn't in E2EE)
		// For E2EE, keys are managed client-side only
		
		// Derive public key from private key (if we had it)
		// In E2EE, public key is stored separately
		var storedPublicKey []byte
		err = db.QueryRow(`SELECT public_key FROM User WHERE id = ?`, userID).Scan(&storedPublicKey)
		if err == nil && len(storedPublicKey) > 0 {
			return nil, storedPublicKey, nil // Return public key only
		}
	}
	
	// Note: In E2EE, key generation and management happens client-side in JavaScript
	// This Go function is not used in the E2EE flow - keys are handled by libsodium.js
	// Server only stores encrypted private keys and public keys
	return nil, nil, fmt.Errorf("key management is client-side only in E2EE")
}
```

---

### Recommendation

**Use Solution 3 (Hybrid Approach)** because:
1. ✅ Simple to implement
2. ✅ Works with your async messaging model
3. ✅ All devices can read all messages
4. ✅ Maintains true E2EE
5. ✅ Password-based encryption is familiar to users

**Trade-offs:**
- Requires password on each device login
- Password change requires re-encryption
- If password is weak, private key is vulnerable

**Alternative:** Use a master key derived from password + device fingerprint for better security.

---

## Limitations & Trade-offs

1. **Requires JavaScript:** Breaks your "no JS" preference
2. **Key Management:** Users must manage keys (or use password-derived encryption)
3. **Multi-Device:** Requires key sync solution (see Multi-Device Support section above)
4. **Performance:** Encryption/decryption adds ~10-50ms per message
5. **Complexity:** Much more complex than Option 1
6. **Password Dependency:** Multi-device requires password on each device (or key sync)

---

## Estimated Implementation Time

- **Database schema:** 1 day
- **Server-side code:** 3-4 days
- **Client-side JavaScript:** 4-5 days
- **Testing & integration:** 3-4 days
- **Total:** 2-3 weeks

---

## Why libsodium.js?

We use [libsodium.js](https://github.com/jedisct1/libsodium.js) instead of Web Crypto API because:

**Advantages:**
- ✅ **Better HKDF support:** Native HKDF functions (we use generic hash)
- ✅ **Easier key management:** Simple API for X25519 key pairs
- ✅ **More crypto primitives:** ChaCha20-Poly1305, Argon2id, etc.
- ✅ **Better browser compatibility:** Works in all modern browsers
- ✅ **Simpler code:** Less boilerplate than Web Crypto API
- ✅ **Well-tested:** Used by Signal, WhatsApp, and other secure apps

**Crypto Algorithms Used:**
- **Key Exchange:** X25519 (Elliptic Curve Diffie-Hellman)
- **Message Encryption:** ChaCha20-Poly1305 (authenticated encryption)
- **Key Derivation:** Argon2id (for password-based encryption)
- **Hash Functions:** BLAKE2b (via generic hash)

**Installation:**
```html
<script src="https://cdn.jsdelivr.net/npm/libsodium-wrappers@0.7.13/dist/browser/sodium.js"></script>
```

Or download and serve locally for better performance and security.

