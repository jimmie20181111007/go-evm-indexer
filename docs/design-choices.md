# Design Decisions

Every choice here was made after running in production and hitting real problems.

---

## 1. Why `(tx_hash, log_index)` and not just `tx_hash`?

A single Ethereum transaction can emit **multiple events**. If you only use `tx_hash` as the dedup key, you'll lose events.

Example:
```
Tx 0x123 emits:
  - log[0]: Deposit(Alice, 1 ETH)
  - log[1]: Transfer(Alice, Bob, 0.5 ETH)
  - log[2]: Deposit(Charlie, 2 ETH)
```

If you dedup by `tx_hash` alone, processing the first log marks the whole tx as "done" — logs 1 and 2 never get processed.

**We use `(chain_id, tx_hash, log_index)` as the unique key.**

---

## 2. Why Redis + DB unique index, not just DB?

**Just DB:**
- Every retry hits Postgres → expensive
- 99% of retries are the same request hitting the API twice
- You're burning DB connections for no reason

**Just Redis:**
- Redis restarts → you lose dedup state
- Race conditions → two requests both pass the check
- Redis isn't the source of truth

**Our approach:**
1. Redis `SETNX` fast-path → 99% of retries are handled in ~1ms
2. DB `UNIQUE INDEX` on `(chain_id, tx_hash, log_index)` → if Redis somehow lets two through, the DB rejects the second one
3. T+1 reconciliation → nightly job that compares processed events with actual chain state

**Why three layers?** Because in production, "it can't happen" always happens eventually.

---

## 3. Why safe head N confirmations behind tip?

Ethereum reorgs happen. Small reorgs (1-2 blocks) are common on busy days.

If you credit a user's deposit at block 12345 and that block gets reorged at 12346:
- User sees "deposit credited" then "deposit disappeared"
- Your accounting is now wrong
- You have a support ticket

**Our rule:**
- Small deposits (< $100): 3 confirmations
- Medium deposits ($100 - $10k): 12 confirmations
- Large deposits (> $10k): 36 confirmations or multi-sig approval

The deeper the confirmation, the less likely a reorg can reach you.

---

## 4. Why cursor-based polling and not `eth_subscribe`?

**`eth_subscribe` sounds nice but:**
- WebSocket connections drop (network blips, node restarts)
- No guarantee you won't miss events during disconnection
- Different nodes have different subscription reliability

**Cursor polling:**
- Dead simple: "what block am I at? give me the next one"
- Crash-safe: if anything dies, just resume from last saved cursor
- Works with any RPC node, no special features needed

**We poll every block and persist the cursor.** Boring, reliable, easy to debug.

---

## 5. Why hot/cold wallet tiers?

You can't just send all user deposits to one wallet:
- One key compromise = total loss
- Hot wallet needs to be online (more attack surface)
- Cold wallet is offline (secure but slow)

**Our approach:**
- **Hot wallet**: small balance, online, used for quick withdrawals
- **Cold wallet**: large balance, offline (MPC multi-sig), moved automatically
- Threshold: anything over $10k goes through multi-sig approval

---

## 6. Why T+1 reconciliation?

No system is perfect. Bugs happen. Race conditions happen.

**T+1 job:**
- Pull all on-chain events for the last 24h
- Compare with what we processed
- Alert on mismatches

This caught bugs that passed all three dedup layers. It's the safety net for the safety nets.
