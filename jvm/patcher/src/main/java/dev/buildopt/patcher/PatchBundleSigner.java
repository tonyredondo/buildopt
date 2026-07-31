package dev.buildopt.patcher;

import java.security.GeneralSecurityException;
import java.security.PrivateKey;
import java.security.Signature;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Canonical Ed25519 signer for a complete unsigned PatchBundle v1 manifest.
 *
 * <p>The signer accepts declarative JSON without {@code bundleDigest} or
 * {@code signature}, calculates the closed digest definition, and returns
 * immutable canonical bytes. The customer-side verifier remains the authority
 * that validates blobs, caller binding, expiration, and all bundle semantics.</p>
 */
public final class PatchBundleSigner {
    private static final int MAXIMUM_MANIFEST_BYTES = 2 * 1024 * 1024;
    private static final Pattern KEY_ID = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$");
    private static final Set<String> UNSIGNED_FIELDS = Set.of(
            "schemaVersion",
            "recordType",
            "contractVersion",
            "repositoryId",
            "actionId",
            "baseRevision",
            "baseTree",
            "sourceStateDigest",
            "recipe",
            "createdAt",
            "expiresAt",
            "operations",
            "blobs",
            "validation",
            "delivery");

    private PatchBundleSigner() {
    }

    /**
     * Signs one complete unsigned manifest without mutating caller bytes.
     *
     * @param unsignedManifest strict UTF-8 JSON without digest or signature
     * @param keyId configured deployment signing-key identity
     * @param privateKey Ed25519 private key held by the control plane
     * @return canonical signed manifest and its content digest
     * @throws PatchFailure when input or signing authority is invalid
     */
    public static SignedPatchBundle sign(
            byte[] unsignedManifest,
            String keyId,
            PrivateKey privateKey) throws PatchFailure {
        try {
            if (unsignedManifest == null
                    || unsignedManifest.length == 0
                    || unsignedManifest.length > MAXIMUM_MANIFEST_BYTES) {
                reject("unsigned manifest size is outside the supported bound");
            }
            if (keyId == null || !KEY_ID.matcher(keyId).matches()) {
                reject("signing keyId is invalid");
            }
            if (privateKey == null || !"EdDSA".equals(privateKey.getAlgorithm())) {
                reject("configured signing key is not Ed25519");
            }
            Object parsed = StrictJson.parse(unsignedManifest.clone());
            if (!(parsed instanceof Map<?, ?>)) {
                reject("unsigned manifest is not an object");
            }
            Map<?, ?> raw = (Map<?, ?>) parsed;
            Map<String, Object> root = new LinkedHashMap<>();
            for (Map.Entry<?, ?> entry : raw.entrySet()) {
                if (!(entry.getKey() instanceof String)) {
                    reject("unsigned manifest field is not a string");
                }
                String field = (String) entry.getKey();
                root.put(field, entry.getValue());
            }
            if (!root.keySet().equals(UNSIGNED_FIELDS)) {
                reject("unsigned manifest fields are incomplete or unknown");
            }
            if (!"1.0".equals(root.get("schemaVersion"))
                    || !"PATCH_BUNDLE".equals(root.get("recordType"))
                    || !"buildopt-patch-bundle/v1".equals(root.get("contractVersion"))) {
                reject("unsigned manifest identity is invalid");
            }

            String bundleDigest = PatchBundleVerifier.calculateBundleDigest(root);
            root.put("bundleDigest", bundleDigest);
            Map<String, Object> authentication = new LinkedHashMap<>();
            authentication.put("algorithm", "Ed25519");
            authentication.put("canonicalization", "JCS");
            authentication.put("keyId", keyId);
            authentication.put("signedBundleDigest", bundleDigest);
            Signature signer = Signature.getInstance("Ed25519");
            signer.initSign(privateKey);
            signer.update(PatchBundleVerifier.signaturePayload(
                    bundleDigest,
                    "buildopt-patch-bundle/v1",
                    keyId));
            authentication.put(
                    "value",
                    Base64.getUrlEncoder().withoutPadding().encodeToString(signer.sign()));
            root.put("signature", authentication);
            return new SignedPatchBundle(bundleDigest, StrictJson.canonicalBytes(root));
        } catch (PatchFailure failure) {
            throw failure;
        } catch (GeneralSecurityException | IllegalArgumentException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    "bundle signing failed: " + exception.getMessage(),
                    exception);
        }
    }

    private static void reject(String message) throws PatchFailure {
        throw new PatchFailure(PatchFailure.Status.REJECTED, message);
    }

    /** Immutable signed output; manifest bytes are always defensively copied. */
    public static final class SignedPatchBundle {
        private final String bundleDigest;
        private final byte[] canonicalManifest;

        private SignedPatchBundle(String bundleDigest, byte[] canonicalManifest) {
            this.bundleDigest = bundleDigest;
            this.canonicalManifest = canonicalManifest.clone();
        }

        public String bundleDigest() {
            return bundleDigest;
        }

        public byte[] canonicalManifest() {
            return canonicalManifest.clone();
        }
    }
}
