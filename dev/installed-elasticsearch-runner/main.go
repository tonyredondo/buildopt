// Command installed-elasticsearch-runner freezes the EIC allocation, checks
// budget estimates, qualifies local capture, and supervises native preparation.
// Candidate execution remains gated by its additional runtime proof.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "owner-test-child" {
		os.Exit(ownerTestChild(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "correctness-child" {
		os.Exit(correctnessChild(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "mutation-child" {
		os.Exit(mutationChild(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "native-child" {
		os.Exit(nativeChild(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "fixture-child" {
		os.Exit(fixtureChild(os.Args[2:]))
	}
	if err := runCLI(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCLI(args []string, out io.Writer) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return errors.New("EIC protocol capture requires Linux AMD64")
	}
	if len(args) == 0 {
		return errors.New("mode required: " + runnerModes)
	}
	mode := args[0]
	flags := flag.NewFlagSet(mode, flag.ContinueOnError)
	var root, pkg, pin, input, cache, jdkArchive, slot string
	var charged, ceiling int64
	var first, second fileBinding
	var metadata fileBinding
	var source, gradle, jdk string
	switch mode {
	case "mutation-freeze":
		flags.StringVar(&root, "root", "", "new absolute local qualification root")
		flags.StringVar(&source, "source", "", "clean frozen B0 worktree")
		flags.StringVar(&gradle, "gradle", "", "locked Gradle distribution directory")
		flags.StringVar(&jdk, "jdk", "", "locked JDK directory")
		flags.StringVar(&first.Path, "correctness-receipt", "", "optional verified C receipt for the separate public M allocation")
		flags.StringVar(&first.SHA256, "correctness-sha256", "", "external C receipt digest")
	case "mutation-run", "mutation-check", "testkit-count", "boundary-check", "correctness-run", "correctness-check", "owner-test-freeze", "owner-test-continue", "owner-test-run", "owner-test-check":
		flags.StringVar(&first.Path, "input", "", "absolute frozen input or receipt")
		flags.StringVar(&first.SHA256, "sha256", "", "externally retained input digest")
		if mode == "owner-test-freeze" {
			flags.StringVar(&second.Path, "composite-decision", "", "accepted composite owner-test decision")
			flags.StringVar(&second.SHA256, "composite-decision-sha256", "", "external decision digest")
		}
	case "native-compare", "native-output-audit", "metadata-compare":
		flags.StringVar(&first.Path, "first", "", "absolute first diagnostic-complete.json")
		flags.StringVar(&first.SHA256, "first-sha256", "", "external first capture digest")
		flags.StringVar(&second.Path, "second", "", "absolute second diagnostic-complete.json")
		flags.StringVar(&second.SHA256, "second-sha256", "", "external second capture digest")
		if mode == "metadata-compare" {
			flags.StringVar(&metadata.Path, "metadata", "", "approved and qualified metadata runtime")
			flags.StringVar(&metadata.SHA256, "metadata-sha256", "", "external metadata runtime digest")
		}
	case "clean-seal":
		flags.StringVar(&root, "root", "", "absolute clean campaign root")
	case "native-prepare", "native-diagnose", "native-clean-diagnose":
		flags.StringVar(&root, "root", "", "absolute native campaign root")
		flags.StringVar(&slot, "slot", "", "frozen native preparation or diagnostic slot")
		if mode == "native-clean-diagnose" {
			flags.StringVar(&pin, "seed-sha256", "", "external frozen clean seed digest")
		}
	case "plan", "correctness-plan":
	case "audit":
		flags.StringVar(&cache, "cache", "", "absolute retained static source/history cache")
		flags.StringVar(&pkg, "package", "", "absolute package path")
		flags.StringVar(&jdkArchive, "jdk-archive", "", "absolute locked Temurin archive")
	case "budget":
		flags.Int64Var(&ceiling, "ceiling-seconds", 0, "explicit owner-approved total planning envelope; zero preserves historical 7200")
		flags.StringVar(&input, "input", "", "absolute JSON estimate input; includes accumulated costs")
		flags.Int64Var(&charged, "charged-seconds", 0, "accumulated task seconds; at least 4440, no new window")
	case "fixture":
		flags.StringVar(&root, "root", "", "new absolute local fixture output directory")
		flags.StringVar(&pkg, "package", "", "absolute package path, bound by its actual bytes")
	case "check":
		flags.StringVar(&root, "root", "", "absolute retained fixture directory")
		flags.StringVar(&pin, "receipt-sha256", "", "external SHA-256 printed by fixture capture")
	default:
		return errors.New("unknown mode; use " + runnerModes)
	}
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	switch mode {
	case "metadata-compare":
		session, err := newMetadataSession(&metadata)
		if err != nil {
			return err
		}
		contract, err := loadAcceptedDateContract()
		if err != nil {
			return err
		}
		result, err := compareOutputs(first, second, contract, session)
		if err != nil {
			return err
		}
		return encoder.Encode(result)
	case "native-output-audit":
		result, err := auditNativeOutputs(first, second)
		if err != nil {
			return err
		}
		return encoder.Encode(result)
	case "owner-test-continue":
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		binding, err := freezeOwnerContinuation(ctx, first)
		if err != nil {
			return err
		}
		return encoder.Encode(binding)
	case "owner-test-freeze":
		// The public M gate also replays every retained C output before setup.
		// This bounds preparation only; native T keeps its own frozen deadline.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		var decision *fileBinding
		if second.Path != "" || second.SHA256 != "" {
			decision = &second
		}
		binding, err := freezeOwnerTests(ctx, first, decision)
		if err != nil {
			return err
		}
		return encoder.Encode(binding)
	case "owner-test-run":
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		binding, err := runOwnerTests(ctx, first)
		return errors.Join(err, encoder.Encode(binding))
	case "owner-test-check":
		result, err := checkOwnerTests(first)
		if err != nil {
			return err
		}
		return encoder.Encode(result)
	case "correctness-run":
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		binding, err := runCorrectness(ctx, first)
		return errors.Join(err, encoder.Encode(binding))
	case "correctness-check":
		result, err := checkCorrectness(first)
		if err != nil {
			return err
		}
		return encoder.Encode(result)
	case "boundary-check":
		result, err := checkInstalledBoundaries(first)
		if err != nil {
			return err
		}
		return encoder.Encode(result)
	case "testkit-count":
		result, err := countTestKitBuilds(first)
		if e := encoder.Encode(result); e != nil {
			return errors.Join(err, e)
		}
		return err
	case "mutation-freeze":
		limit := 30 * time.Second
		var parent *fileBinding
		if first.Path != "" || first.SHA256 != "" {
			parent = &first
			limit = 180 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		defer cancel()
		binding, err := freezeMutationAllocation(ctx, root, source, gradle, jdk, parent)
		if err != nil {
			return err
		}
		return encoder.Encode(binding)
	case "mutation-run":
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		binding, err := runMutations(ctx, first)
		if e := encoder.Encode(binding); e != nil {
			return errors.Join(err, e)
		}
		return err
	case "mutation-check":
		summary, err := checkMutations(first)
		if err != nil {
			return err
		}
		return encoder.Encode(summary)
	case "native-compare":
		contract, err := loadAcceptedDateContract()
		if err != nil {
			return err
		}
		result, err := compareNativeOutputs(first, second, contract)
		if err != nil {
			return err
		}
		result.ContractSHA256 = digest(acceptedDateContract)
		return encoder.Encode(result)
	case "clean-seal":
		seed, err := currentCleanSeed(root)
		if err != nil {
			return err
		}
		path := filepath.Join(root, "clean-seed.json")
		if err := writeJSON(path, seed); err != nil {
			return err
		}
		pin, err := hashFile(path)
		if err != nil {
			return err
		}
		return encoder.Encode(fileBinding{Path: path, SHA256: pin})
	case "native-prepare", "native-diagnose", "native-clean-diagnose":
		if (mode == "native-prepare" && slot != "P001" && slot != "P002" && slot != "P003") || (mode == "native-diagnose" && slot != "D001" && slot != "D002") || (mode == "native-clean-diagnose" && slot != "CD001" && slot != "CD002") {
			return errors.New("slot does not belong to selected native mode")
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		r, err := nativePrepareWithSeed(ctx, root, slot, pin)
		if encodeErr := encoder.Encode(r); encodeErr != nil {
			return errors.Join(err, encodeErr)
		}
		return err
	case "correctness-plan":
		return encoder.Encode(makeCorrectnessPlan())
	case "plan":
		return encoder.Encode(makeProtocol())
	case "audit":
		a, err := auditStatic(cache, pkg, jdkArchive)
		if err != nil {
			return err
		}
		return encoder.Encode(a)
	case "budget":
		b := budgetInput{ChargedSeconds: charged, CeilingSeconds: ceiling, Estimates: []costEstimate{}}
		if input != "" {
			if charged != 0 || ceiling != 0 {
				return errors.New("use input or explicit budget flags, not both")
			}
			if err := readJSON(input, &b); err != nil {
				return err
			}
		}
		r, err := admit(makeProtocol(), b)
		if err != nil {
			return err
		}
		return encoder.Encode(r)
	case "check":
		s, err := checkCapture(root, pin)
		if err != nil {
			return err
		}
		return encoder.Encode(s)
	case "fixture":
		executable, err := ownExecutable()
		if err != nil {
			return err
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		pin, err := runFixture(ctx, root, executable, []string{"fixture-child"}, pkg, 120*time.Second)
		if pin != "" {
			if encodeErr := encoder.Encode(struct {
				ReceiptSHA256    string `json:"receiptSha256"`
				EnvironmentClass string `json:"environmentClass"`
			}{pin, fixtureClass}); encodeErr != nil {
				return errors.Join(err, encodeErr)
			}
		}
		return err
	}
	return errors.New("unreachable mode")
}

const runnerModes = "plan, correctness-plan, correctness-run, correctness-check, owner-test-freeze, owner-test-continue, owner-test-run, owner-test-check, budget, audit, fixture, check, native-prepare, native-diagnose, clean-seal, native-clean-diagnose, native-compare, native-output-audit, metadata-compare, mutation-freeze, mutation-run, mutation-check, testkit-count, boundary-check"
