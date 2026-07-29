package dev.buildopt.patcher;

import static java.nio.file.LinkOption.NOFOLLOW_LINKS;
import static java.nio.file.StandardCopyOption.ATOMIC_MOVE;
import static java.nio.file.StandardCopyOption.REPLACE_EXISTING;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.attribute.BasicFileAttributes;
import java.nio.file.attribute.PosixFilePermission;
import java.nio.file.attribute.PosixFilePermissions;
import java.security.GeneralSecurityException;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;

import dev.buildopt.patcher.PatchBundleVerifier.Operation;
import dev.buildopt.patcher.PatchBundleVerifier.VerifiedBundle;
import dev.buildopt.patcher.PatchBundleVerifier.VerifiedPatchBundle;

/**
 * Exact PatchBundle materializer for a private detached Git worktree.
 *
 * <p>The applier never executes bundle content, hooks, a fuzzy patch, a
 * rebase, or a force update. It creates only an absent action branch after
 * all staged postimages match.</p>
 */
public final class PatchBundleApplier {
    private static final Set<PosixFilePermission> PRIVATE_DIRECTORY =
            PosixFilePermissions.fromString("rwx------");
    private static final Set<PosixFilePermission> PRIVATE_FILE =
            PosixFilePermissions.fromString("rw-------");
    private static final Set<PosixFilePermission> SOURCE_FILE =
            PosixFilePermissions.fromString("rw-r--r--");

    /** Successful delivery outcomes. */
    public enum Outcome {
        DRAFT_PR_CREATED,
        EXISTING_DRAFT_PR
    }

    /** Test-only bounded fault points used by the executable spike. */
    enum Fault {
        NONE,
        CORRUPT_POSTIMAGE,
        INTERRUPT_AFTER_WRITE
    }

    /**
     * Applies a verified bundle and creates or recovers its draft delivery.
     *
     * @param bundle strictly verified bundle
     * @param repository customer Git worktree
     * @param stagingParent private directory outside the customer repository
     * @param pullRequests draft-PR delivery adapter owned by the caller
     * @return created or replayed draft delivery
     * @throws PatchFailure on any fail-closed result
     */
    public Result apply(
            VerifiedPatchBundle bundle,
            Path repository,
            Path stagingParent,
            DraftPullRequests pullRequests) throws PatchFailure {
        return apply(bundle, repository, stagingParent, pullRequests, Fault.NONE);
    }

    Result apply(
            VerifiedPatchBundle verified,
            Path repository,
            Path stagingParent,
            DraftPullRequests pullRequests,
            Fault fault) throws PatchFailure {
        if (!(verified instanceof VerifiedBundle bundle)) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    "unrecognized verified bundle implementation");
        }
        Path root = verifyRepository(bundle, repository);
        Path privateParent = prepareStagingParent(root, stagingParent);
        Path staging = createUnusedStagingPath(privateParent);
        boolean worktreeAdded = false;
        try {
            gitRequired(root, List.of(
                    "worktree",
                    "add",
                    "--detach",
                    "--no-checkout",
                    staging.toString(),
                    bundle.baseRevision()));
            worktreeAdded = true;
            try {
                setPermissions(staging, PRIVATE_DIRECTORY);
            } catch (IOException exception) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "cannot make the staging worktree private",
                        exception);
            }
            gitRequired(staging, List.of("read-tree", bundle.baseRevision()));
            verifyStagingBase(bundle, staging);
            for (Operation operation : bundle.operations()) {
                applyOperation(bundle, root, staging, operation, fault);
            }
            verifyStagedPaths(bundle, staging);
            if (fault == Fault.INTERRUPT_AFTER_WRITE) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "injected interruption before commit");
            }

            String tree = gitTextRequired(staging, List.of("write-tree"));
            String commit = createCommit(bundle, root, tree);
            String branch = "buildopt/" + bundle.actionId();
            String branchRef = "refs/heads/" + branch;
            String existing = readRef(root, branchRef);
            if (existing.isEmpty()) {
                GitResult creation = git(
                        root,
                        List.of(
                                "update-ref",
                                branchRef,
                                commit,
                                "0".repeat(commit.length())),
                        Map.of(),
                        null);
                if (creation.exitCode() != 0) {
                    existing = readRef(root, branchRef);
                    if (existing.isEmpty()) {
                        throw new PatchFailure(
                                PatchFailure.Status.PROPOSED,
                                "atomic action branch creation failed: "
                                        + creation.outputText().trim());
                    }
                } else {
                    existing = commit;
                }
            }
            if (!existing.equals(commit)) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "existing action branch differs from the exact candidate");
            }

            Identity identity = new Identity(
                    bundle.repositoryId(),
                    bundle.actionId(),
                    bundle.bundleDigest());
            Optional<DraftPullRequest> current;
            try {
                current = pullRequests.find(identity);
            } catch (IOException exception) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "draft PR lookup failed after exact branch recovery",
                        exception);
            }
            if (current.isPresent()) {
                DraftPullRequest pullRequest = current.orElseThrow();
                if (!pullRequest.draft()
                        || !pullRequest.branch().equals(branch)
                        || !pullRequest.headCommit().equals(commit)) {
                    throw new PatchFailure(
                            PatchFailure.Status.PROPOSED,
                            "existing PR does not match the immutable action delivery");
                }
                return new Result(
                        Outcome.EXISTING_DRAFT_PR,
                        branch,
                        commit,
                        tree);
            }
            try {
                pullRequests.create(new DraftPullRequest(
                        identity,
                        branch,
                        commit,
                        true));
            } catch (IOException exception) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "action branch is intact but draft PR creation failed",
                        exception);
            }
            return new Result(
                    Outcome.DRAFT_PR_CREATED,
                    branch,
                    commit,
                    tree);
        } finally {
            cleanupStaging(root, staging, worktreeAdded);
        }
    }

    /**
     * Computes the exact source-state digest used by this bounded spike.
     *
     * <p>The digest covers the NUL-delimited recursive Git tree inventory at
     * the exact base revision.</p>
     */
    public static String sourceStateDigest(Path repository, String baseRevision)
            throws PatchFailure {
        try {
            Path root = repository.toRealPath(NOFOLLOW_LINKS);
            byte[] inventory = gitRequired(
                    root,
                    List.of("ls-tree", "-r", "-z", "--full-tree", baseRevision))
                    .output();
            return PatchBundleVerifier.digestBytes(inventory);
        } catch (IOException | GeneralSecurityException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "cannot calculate exact source-state digest",
                    exception);
        }
    }

    private static Path verifyRepository(VerifiedBundle bundle, Path repository)
            throws PatchFailure {
        try {
            Path root = repository.toRealPath(NOFOLLOW_LINKS);
            if (!Files.isDirectory(root, NOFOLLOW_LINKS)) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "repository is not a directory");
            }
            Path reportedRoot = Path.of(gitTextRequired(
                    root,
                    List.of("rev-parse", "--show-toplevel"))).toRealPath(NOFOLLOW_LINKS);
            if (!root.equals(reportedRoot)) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "repository path is not the Git worktree root");
            }
            String revision = gitTextRequired(
                    root,
                    List.of(
                            "rev-parse",
                            "--verify",
                            bundle.baseRevision() + "^{commit}"));
            if (!revision.equals(bundle.baseRevision())) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "base revision is not the exact requested commit");
            }
            String tree = gitTextRequired(
                    root,
                    List.of("rev-parse", bundle.baseRevision() + "^{tree}"));
            if (!tree.equals(bundle.baseTree())) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "base tree does not match the signed bundle");
            }
            String sourceState = sourceStateDigest(root, bundle.baseRevision());
            if (!sourceState.equals(bundle.sourceStateDigest())) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "source-state digest does not match the signed bundle");
            }
            return root;
        } catch (IOException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "cannot verify repository: " + exception.getMessage(),
                    exception);
        }
    }

    private static Path prepareStagingParent(Path repository, Path requested)
            throws PatchFailure {
        try {
            Files.createDirectories(requested);
            Path parent = requested.toRealPath(NOFOLLOW_LINKS);
            if (!Files.isDirectory(parent, NOFOLLOW_LINKS)
                    || parent.startsWith(repository)) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "staging parent must be an external directory");
            }
            setPermissions(parent, PRIVATE_DIRECTORY);
            return parent;
        } catch (IOException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "cannot prepare private staging parent",
                    exception);
        }
    }

    private static Path createUnusedStagingPath(Path parent) throws PatchFailure {
        try {
            Path path = Files.createTempDirectory(parent, "candidate-");
            setPermissions(path, PRIVATE_DIRECTORY);
            Files.delete(path);
            return path;
        } catch (IOException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "cannot reserve private staging path",
                    exception);
        }
    }

    private static void verifyStagingBase(VerifiedBundle bundle, Path staging)
            throws PatchFailure {
        String revision = gitTextRequired(staging, List.of("rev-parse", "HEAD"));
        String tree = gitTextRequired(staging, List.of("rev-parse", "HEAD^{tree}"));
        if (!revision.equals(bundle.baseRevision()) || !tree.equals(bundle.baseTree())) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "detached staging worktree is not at the signed base");
        }
    }

    private static void applyOperation(
            VerifiedBundle bundle,
            Path repository,
            Path staging,
            Operation operation,
            Fault fault) throws PatchFailure {
        validateTargetGraph(bundle, repository, staging, operation.path());
        Path target = staging.resolve(operation.path()).normalize();
        boolean modify = "MODIFY".equals(operation.type());
        if (modify) {
            IndexEntry entry = requireTrackedRegularFile(staging, operation.path());
            byte[] preimageBytes = gitRequired(
                    staging,
                    List.of("cat-file", "blob", entry.objectId())).output();
            String preimage;
            try {
                preimage = PatchBundleVerifier.digestBytes(preimageBytes);
            } catch (GeneralSecurityException exception) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "cannot digest exact Git preimage",
                        exception);
            }
            if (!preimage.equals(operation.preimageDigest())) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "exact preimage differs for " + operation.path());
            }
        } else if (Files.exists(target, NOFOLLOW_LINKS)
                || indexEntry(staging, operation.path()).isPresent()) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "ADD target already exists: " + operation.path());
        }

        Path parent = target.getParent();
        createLinkSafeDirectories(staging, parent);
        Path temporary = null;
        try {
            temporary = Files.createTempFile(parent, ".buildopt-", ".tmp");
            setPermissions(temporary, PRIVATE_FILE);
            Files.write(temporary, operation.blob().content());
            if (!digestFile(temporary, PatchFailure.Status.UNCHANGED)
                    .equals(operation.postimageDigest())) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "temporary replacement digest differs");
            }
            setPermissions(temporary, SOURCE_FILE);
            if (modify) {
                Files.move(temporary, target, ATOMIC_MOVE, REPLACE_EXISTING);
            } else {
                Files.move(temporary, target, ATOMIC_MOVE);
            }
            temporary = null;
            setPermissions(target, SOURCE_FILE);
            if (fault == Fault.CORRUPT_POSTIMAGE) {
                Files.writeString(
                        target,
                        "injected-corruption",
                        StandardCharsets.UTF_8);
            }
            if (!digestFile(target, PatchFailure.Status.UNCHANGED)
                    .equals(operation.postimageDigest())) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "postimage differs for " + operation.path());
            }
            GitResult hashed = git(
                    staging,
                    List.of("hash-object", "-w", "--stdin"),
                    Map.of(),
                    operation.blob().content());
            if (hashed.exitCode() != 0) {
                throw unchanged(
                        "cannot persist exact replacement blob",
                        hashed.outputText());
            }
            String objectId = hashed.outputText().trim();
            gitRequired(staging, List.of(
                    "update-index",
                    "--add",
                    "--cacheinfo",
                    "100644," + objectId + "," + operation.path()));
            Optional<IndexEntry> staged = indexEntry(staging, operation.path());
            if (staged.isEmpty()
                    || !"100644".equals(staged.orElseThrow().mode())
                    || !objectId.equals(staged.orElseThrow().objectId())) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "staged result is not the exact mode-100644 blob");
            }
        } catch (IOException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "cannot atomically materialize " + operation.path(),
                    exception);
        } finally {
            if (temporary != null) {
                try {
                    Files.deleteIfExists(temporary);
                } catch (IOException ignored) {
                    // The private worktree is removed in the outer finally block.
                }
            }
        }
    }

    private static void validateTargetGraph(
            VerifiedBundle bundle,
            Path repository,
            Path staging,
            String relative) throws PatchFailure {
        PatchBundleVerifier.safePath(relative, "operation path");
        Path target = staging.resolve(relative).normalize();
        if (!target.startsWith(staging)) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    "operation path escapes staging");
        }
        String[] segments = relative.split("/", -1);
        Path current = staging;
        StringBuilder gitPath = new StringBuilder();
        for (int index = 0; index < segments.length; index++) {
            if (index > 0) {
                gitPath.append('/');
            }
            gitPath.append(segments[index]);
            current = current.resolve(segments[index]);
            String treeEntry = gitText(
                    repository,
                    List.of(
                            "ls-tree",
                            bundle.baseRevision(),
                            "--",
                            gitPath.toString()));
            if (treeEntry.startsWith("160000 ")) {
                throw new PatchFailure(
                        PatchFailure.Status.REJECTED,
                        "operation path enters a Git submodule");
            }
            if (treeEntry.startsWith("120000 ")) {
                throw new PatchFailure(
                        PatchFailure.Status.REJECTED,
                        "operation path contains a Git symlink");
            }
            if (Files.exists(current, NOFOLLOW_LINKS)) {
                BasicFileAttributes attributes;
                try {
                    attributes = Files.readAttributes(
                            current,
                            BasicFileAttributes.class,
                            NOFOLLOW_LINKS);
                } catch (IOException exception) {
                    throw new PatchFailure(
                            PatchFailure.Status.REJECTED,
                            "cannot inspect operation path graph",
                            exception);
                }
                if (attributes.isSymbolicLink()) {
                    throw new PatchFailure(
                            PatchFailure.Status.REJECTED,
                            "operation path contains a symlink");
                }
                if (index < segments.length - 1
                        && !attributes.isDirectory()) {
                    throw new PatchFailure(
                            PatchFailure.Status.REJECTED,
                            "operation parent is not a directory");
                }
                if (index < segments.length - 1
                        && Files.exists(current.resolve(".git"), NOFOLLOW_LINKS)) {
                    throw new PatchFailure(
                            PatchFailure.Status.REJECTED,
                            "operation path enters a nested repository");
                }
            }
        }
    }

    private static IndexEntry requireTrackedRegularFile(
            Path staging,
            String relative) throws PatchFailure {
        Optional<IndexEntry> entry = indexEntry(staging, relative);
        if (entry.isEmpty()) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "MODIFY target is absent from the exact base index");
        }
        if (!"100644".equals(entry.orElseThrow().mode())) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    "MODIFY target is not tracked at mode 100644");
        }
        return entry.orElseThrow();
    }

    private static Optional<IndexEntry> indexEntry(
            Path staging,
            String relative) throws PatchFailure {
        byte[] output = gitRequired(
                staging,
                List.of("ls-files", "--stage", "-z", "--", relative)).output();
        if (output.length == 0) {
            return Optional.empty();
        }
        if (output[output.length - 1] != 0) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "Git index entry is not NUL terminated");
        }
        String value = new String(
                output,
                0,
                output.length - 1,
                StandardCharsets.UTF_8);
        int tab = value.indexOf('\t');
        String[] metadata = tab < 0
                ? new String[0]
                : value.substring(0, tab).split(" ", -1);
        String path = tab < 0 ? "" : value.substring(tab + 1);
        if (metadata.length != 3 || !path.equals(relative)) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "Git index returned an ambiguous entry");
        }
        return Optional.of(new IndexEntry(metadata[0], metadata[1]));
    }

    private static void createLinkSafeDirectories(Path root, Path parent)
            throws PatchFailure {
        Path relative = root.relativize(parent);
        Path current = root;
        for (Path segment : relative) {
            current = current.resolve(segment);
            try {
                if (Files.exists(current, NOFOLLOW_LINKS)) {
                    BasicFileAttributes attributes = Files.readAttributes(
                            current,
                            BasicFileAttributes.class,
                            NOFOLLOW_LINKS);
                    if (!attributes.isDirectory() || attributes.isSymbolicLink()) {
                        throw new PatchFailure(
                                PatchFailure.Status.REJECTED,
                                "writable parent contains a link or non-directory");
                    }
                } else {
                    Files.createDirectory(current);
                    setPermissions(current, PRIVATE_DIRECTORY);
                }
            } catch (IOException exception) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "cannot create safe operation parent",
                        exception);
            }
        }
    }

    private static void verifyStagedPaths(VerifiedBundle bundle, Path staging)
            throws PatchFailure {
        GitResult check = git(
                staging,
                List.of(
                        "diff",
                        "--no-ext-diff",
                        "--no-textconv",
                        "--cached",
                        "--check"),
                Map.of(),
                null);
        if (check.exitCode() != 0) {
            throw unchanged("staged Git diff check failed", check.outputText());
        }
        byte[] rawNames = gitRequired(
                staging,
                List.of(
                        "diff",
                        "--no-ext-diff",
                        "--no-textconv",
                        "--cached",
                        "--name-only",
                        "-z")).output();
        Set<String> names = new HashSet<>();
        int start = 0;
        for (int index = 0; index < rawNames.length; index++) {
            if (rawNames[index] == 0) {
                names.add(new String(
                        rawNames,
                        start,
                        index - start,
                        StandardCharsets.UTF_8));
                start = index + 1;
            }
        }
        Set<String> expected = new HashSet<>();
        for (Operation operation : bundle.operations()) {
            expected.add(operation.path());
        }
        if (start != rawNames.length || !names.equals(expected)) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "staged path set differs from signed operations");
        }
    }

    private static String createCommit(
            VerifiedBundle bundle,
            Path repository,
            String tree) throws PatchFailure {
        String message = "BuildOpt PatchBundle " + bundle.actionId() + "\n\n"
                + "BuildOpt-Repository: " + bundle.repositoryId() + "\n"
                + "BuildOpt-Action: " + bundle.actionId() + "\n"
                + "BuildOpt-Bundle: " + bundle.bundleDigest() + "\n"
                + "BuildOpt-Source-State: " + bundle.sourceStateDigest() + "\n";
        Map<String, String> environment = new LinkedHashMap<>();
        environment.put("GIT_AUTHOR_NAME", "BuildOpt Patcher");
        environment.put("GIT_AUTHOR_EMAIL", "patcher@buildopt.invalid");
        environment.put("GIT_COMMITTER_NAME", "BuildOpt Patcher");
        environment.put("GIT_COMMITTER_EMAIL", "patcher@buildopt.invalid");
        environment.put("GIT_AUTHOR_DATE", bundle.createdAt().toString());
        environment.put("GIT_COMMITTER_DATE", bundle.createdAt().toString());
        GitResult result = git(
                repository,
                List.of(
                        "commit-tree",
                        tree,
                        "-p",
                        bundle.baseRevision()),
                environment,
                message.getBytes(StandardCharsets.UTF_8));
        if (result.exitCode() != 0) {
            throw unchanged("cannot create detached candidate commit", result.outputText());
        }
        String commit = result.outputText().trim();
        String committedTree = gitTextRequired(
                repository,
                List.of("rev-parse", commit + "^{tree}"));
        if (!committedTree.equals(tree)) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "candidate commit tree differs after persistence");
        }
        return commit;
    }

    private static String readRef(Path repository, String reference)
            throws PatchFailure {
        GitResult result = git(
                repository,
                List.of("rev-parse", "--verify", "--quiet", reference),
                Map.of(),
                null);
        if (result.exitCode() == 1) {
            return "";
        }
        if (result.exitCode() != 0) {
            throw unchanged("cannot inspect action branch", result.outputText());
        }
        return result.outputText().trim();
    }

    private static String digestFile(Path path, PatchFailure.Status status)
            throws PatchFailure {
        try {
            return PatchBundleVerifier.digestBytes(Files.readAllBytes(path));
        } catch (IOException | GeneralSecurityException exception) {
            throw new PatchFailure(status, "cannot digest exact file " + path, exception);
        }
    }

    private static void setPermissions(Path path, Set<PosixFilePermission> permissions)
            throws IOException {
        Files.setPosixFilePermissions(path, permissions);
    }

    private static void cleanupStaging(
            Path repository,
            Path staging,
            boolean worktreeAdded) throws PatchFailure {
        if (worktreeAdded) {
            git(
                    repository,
                    List.of("worktree", "remove", "--force", staging.toString()),
                    Map.of(),
                    null);
        }
        try {
            deleteRecursively(staging);
        } catch (IOException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "cannot remove private staging directory",
                    exception);
        }
        if (Files.exists(staging, NOFOLLOW_LINKS)) {
            throw new PatchFailure(
                    PatchFailure.Status.UNCHANGED,
                    "private staging directory remains after cleanup");
        }
        if (worktreeAdded) {
            GitResult prune = git(
                    repository,
                    List.of("worktree", "prune"),
                    Map.of(),
                    null);
            if (prune.exitCode() != 0) {
                throw unchanged(
                        "cannot prune private staging metadata",
                        prune.outputText());
            }
            String worktrees = gitTextRequired(
                    repository,
                    List.of("worktree", "list", "--porcelain"));
            if (worktrees.lines().anyMatch(
                    line -> line.equals("worktree " + staging))) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "private staging metadata remains after cleanup");
            }
        }
    }

    private static void deleteRecursively(Path path) throws IOException {
        if (!Files.exists(path, NOFOLLOW_LINKS)) {
            return;
        }
        BasicFileAttributes attributes = Files.readAttributes(
                path,
                BasicFileAttributes.class,
                NOFOLLOW_LINKS);
        if (attributes.isDirectory() && !attributes.isSymbolicLink()) {
            try (DirectoryStream<Path> children = Files.newDirectoryStream(path)) {
                for (Path child : children) {
                    deleteRecursively(child);
                }
            }
        }
        Files.deleteIfExists(path);
    }

    private static GitResult gitRequired(Path repository, List<String> arguments)
            throws PatchFailure {
        GitResult result = git(repository, arguments, Map.of(), null);
        if (result.exitCode() != 0) {
            throw unchanged(
                    "Git command failed: " + String.join(" ", arguments),
                    result.outputText());
        }
        return result;
    }

    private static String gitTextRequired(Path repository, List<String> arguments)
            throws PatchFailure {
        return gitRequired(repository, arguments).outputText().trim();
    }

    private static String gitText(Path repository, List<String> arguments)
            throws PatchFailure {
        GitResult result = git(repository, arguments, Map.of(), null);
        if (result.exitCode() != 0) {
            throw unchanged(
                    "Git inspection failed: " + String.join(" ", arguments),
                    result.outputText());
        }
        return result.outputText().trim();
    }

    private static GitResult git(
            Path repository,
            List<String> arguments,
            Map<String, String> environment,
            byte[] input) {
        List<String> command = new ArrayList<>();
        command.add("git");
        command.add("-C");
        command.add(repository.toString());
        command.add("-c");
        command.add("core.hooksPath=/dev/null");
        command.add("-c");
        command.add("core.fsmonitor=false");
        command.addAll(arguments);
        ProcessBuilder builder = new ProcessBuilder(command);
        builder.redirectErrorStream(true);
        builder.environment().put("GIT_CONFIG_NOSYSTEM", "1");
        builder.environment().put("GIT_CONFIG_GLOBAL", "/dev/null");
        builder.environment().put("GIT_NO_REPLACE_OBJECTS", "1");
        builder.environment().put("GIT_PAGER", "cat");
        builder.environment().put("GIT_TERMINAL_PROMPT", "0");
        builder.environment().putAll(environment);
        try {
            Process process = builder.start();
            if (input != null) {
                process.getOutputStream().write(input);
            }
            process.getOutputStream().close();
            ByteArrayOutputStream output = new ByteArrayOutputStream();
            process.getInputStream().transferTo(output);
            int exitCode = process.waitFor();
            return new GitResult(exitCode, output.toByteArray());
        } catch (IOException exception) {
            return new GitResult(
                    127,
                    exception.toString().getBytes(StandardCharsets.UTF_8));
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            return new GitResult(
                    130,
                    exception.toString().getBytes(StandardCharsets.UTF_8));
        }
    }

    private static PatchFailure unchanged(String message, String detail) {
        String suffix = detail.isBlank() ? "" : ": " + detail.trim();
        return new PatchFailure(PatchFailure.Status.UNCHANGED, message + suffix);
    }

    private record GitResult(int exitCode, byte[] output) {
        private GitResult {
            output = output.clone();
        }

        @Override
        public byte[] output() {
            return output.clone();
        }

        private String outputText() {
            return new String(output, StandardCharsets.UTF_8);
        }
    }

    private record IndexEntry(String mode, String objectId) {
    }

    /** Stable idempotency identity. */
    public record Identity(String repositoryId, String actionId, String bundleDigest) {
    }

    /** Minimal immutable draft-PR state owned by the caller's delivery adapter. */
    public record DraftPullRequest(
            Identity identity,
            String branch,
            String headCommit,
            boolean draft) {
    }

    /** Remote workflow boundary; the Java patcher never reads credentials. */
    public interface DraftPullRequests {
        Optional<DraftPullRequest> find(Identity identity) throws IOException;

        void create(DraftPullRequest pullRequest) throws IOException;
    }

    /** Successful immutable branch/draft-PR state. */
    public record Result(
            Outcome outcome,
            String branch,
            String headCommit,
            String tree) {
    }
}
