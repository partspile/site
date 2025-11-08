# Message Encryption Implementation Plan

## Current State
Messages are currently stored in plaintext in the `Message` table. The `content` field contains unencrypted text that can be read by anyone with database access.

## Encryption Options

### Option 1: Server-Side Encryption (Recommended for HTMX/No-JS)

**What it provides:**
- Messages encrypted at rest in the database
- Protection against database breaches
- No JavaScript required
- Works with existing HTMX architecture

**What it doesn't provide:**
- True end-to-end encryption (server can decrypt)
- Protection against server compromise
- Forward secrecy

**Implementation Requirements:**

1. **Database Schema Changes:**
   - Add `encrypted_content` BLOB field to `Message` table
   - Keep `content` field for backward compatibility during migration
   - Add `encryption_key_id` INTEGER field to track which key was used
   - Optionally add `encrypted_at` DATETIME field

2. **Encryption Package:**
   - Create `messaging/encryption.go` package
   - Use AES-256-GCM (authenticated encryption)
   - Generate per-conversation encryption keys
   - Derive keys from conversation ID + master key

3. **Key Management:**
   - Master encryption key stored in config (like JWT secret)
   - Per-conversation keys derived using HKDF
   - Keys never stored in database (derived on-the-fly)

4. **Code Changes:**
   - Modify `AddMessage()` to encrypt before storing
   - Modify `GetMessages()` to decrypt when retrieving
   - Update all message retrieval queries
   - Handle migration of existing messages

5. **Dependencies:**
   - Already have `golang.org/x/crypto` (includes AES, HKDF)
   - No new dependencies needed

**Estimated Effort:** 2-3 days

---

### Option 2: True End-to-End Encryption (Telegram-style)

**What it provides:**
- True E2EE - server never sees plaintext
- Protection against server compromise
- Forward secrecy (with proper implementation)
- Similar security to Telegram Secret Chats

**What it requires:**
- Client-side JavaScript for encryption/decryption
- Key exchange protocol (Diffie-Hellman or X3DH)
- Key management on client devices
- More complex implementation

**Implementation Requirements:**

1. **Client-Side JavaScript:**
   - Add minimal JS library (e.g., Web Crypto API or libsodium.js)
   - Encrypt messages before form submission
   - Decrypt messages after receiving from server
   - Manage encryption keys in browser storage

2. **Key Exchange:**
   - Implement key exchange when conversation starts
   - Use Diffie-Hellman or X3DH protocol
   - Store public keys, derive shared secret
   - Implement forward secrecy (ratchet keys)

3. **Database Schema:**
   - Change `content` to BLOB (store encrypted bytes)
   - Add `encryption_version` field
   - Add `key_exchange_id` field
   - Store public keys in separate table

4. **Server Changes:**
   - Accept encrypted content (don't decrypt)
   - Store encrypted messages as-is
   - Return encrypted messages to clients
   - Handle key exchange endpoints

5. **Dependencies:**
   - JavaScript crypto library (libsodium.js or similar)
   - Server-side key exchange handling
   - More complex error handling

**Estimated Effort:** 2-3 weeks

---

## Recommended Approach: Option 1 (Server-Side Encryption)

Given your HTMX/no-JS preference, I recommend **Option 1** with the following implementation:

### Implementation Details

1. **Encryption Algorithm:**
   - AES-256-GCM (Galois/Counter Mode)
   - Provides authenticated encryption (prevents tampering)
   - Nonce (IV) generated per message

2. **Key Derivation:**
   - Master key in config (32 bytes, separate from JWT secret)
   - Per-conversation key: `HKDF(conversation_id, master_key)`
   - Keys derived on-the-fly, never stored

3. **Message Format:**
   - Encrypted: `nonce (12 bytes) + ciphertext + tag (16 bytes)`
   - Base64 encoded for storage in TEXT field
   - Or store as BLOB

4. **Migration Strategy:**
   - Add new `encrypted_content` field
   - Encrypt existing messages in background
   - Support both encrypted and unencrypted during transition
   - Remove `content` field after migration complete

### Security Considerations

- **Master Key:** Must be kept secret, rotated periodically
- **Key Derivation:** Use HKDF-SHA256 for key derivation
- **Nonce:** Must be unique per message (use crypto/rand)
- **Backup:** Encrypted messages require master key to decrypt
- **Performance:** Minimal overhead (~1ms per encrypt/decrypt)

### Files to Modify

1. `schema.sql` - Add encryption fields
2. `messaging/messaging.go` - Add encryption/decryption
3. `messaging/encryption.go` - New encryption package
4. `config/config.go` - Add encryption key config
5. `handlers/messaging.go` - Ensure encryption in handlers
6. Migration script for existing messages

---

## Comparison with Telegram

**Telegram Secret Chats:**
- Uses MTProto 2.0 protocol
- AES-256 encryption
- Diffie-Hellman key exchange
- Forward secrecy
- Client-side encryption

**Our Option 1:**
- AES-256-GCM encryption
- Server-side encryption
- Per-conversation keys
- No forward secrecy
- Simpler implementation

**Our Option 2:**
- Similar to Telegram Secret Chats
- Requires JavaScript
- More complex but true E2EE

---

## Recommendation

Start with **Option 1** (server-side encryption) because:
1. Works with your HTMX/no-JS architecture
2. Provides good security for most use cases
3. Can be implemented quickly
4. Can be upgraded to Option 2 later if needed

If you need true E2EE later, you can migrate to Option 2, but Option 1 will protect against the most common threat (database breaches).

