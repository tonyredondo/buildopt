# F0-015 Test Optimization contract fixtures

Synthetic `TestCacheGrant` and `TestValidationResult` records.

- Valid records carry a closed Ed25519/JCS envelope and bind repository,
  revision/source state, policy, expiration, selectors, permissions, action,
  and content-addressed artifacts.
- Schema negatives reject a missing signature, wildcard task authority,
  grants with no read/write capability, and a `PASSED` result that carries an
  inconclusive reason.
- Semantic negatives reject a grant whose time window is reversed and a result
  that rebinds the candidate artifact from the originating F0-014 request.

The checker links the grant digest/expiration to the F0-013 policy and links
the result to the F0-014 validation request. Cryptographic verification of the
synthetic signatures and canonical payload digests remains with `F0-020`.
