package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type subjectInputs struct {
	Repository          string `json:"repository"`
	TaskPath            string `json:"taskPath"`
	TaskPreimageSHA256  string `json:"taskPreimageSha256"`
	TaskPostimageSHA256 string `json:"taskPostimageSha256"`
	OwnerInputPath      string `json:"ownerInputPath"`
	OwnerInputSHA256    string `json:"ownerInputSha256"`
	BenignSHA256        string `json:"benignSha256"`
	NegativeSHA256      string `json:"negativeSha256"`
	GradleVersion       string `json:"gradleVersion"`
	GradleArchiveSHA256 string `json:"gradleArchiveSha256"`
	JDKVersion          string `json:"jdkVersion"`
	JDKArchiveSHA256    string `json:"jdkArchiveSha256"`
}

func sourceInputs() subjectInputs {
	return subjectInputs{
		Repository:          "https://github.com/elastic/elasticsearch",
		TaskPath:            "build-tools-internal/src/main/java/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsTask.java",
		TaskPreimageSHA256:  "61fe2eaa06ff463c2a49cea656b147889060855acfa094288ebb8b31567e11b6",
		TaskPostimageSHA256: "d6858f5ac43ad671496cf1e54e7af309cb98ebda4a9be53e61578df8baefa2d0",
		OwnerInputPath:      "server/src/main/java/org/elasticsearch/indices/recovery/RecoveryState.java",
		OwnerInputSHA256:    "09c15178d0ebecb3da1d2efccd73d2e40689d060409c196a7ec2ca2f18c4d886",
		BenignSHA256:        "1a35aa9d75c9e922c783ce3bde85e700828440d4ccd0286944575547091269c0",
		NegativeSHA256:      "9ca0190189574e0aeac1aa08111bf5c20e5f2c5246d02af99a91205ab6ba70f1",
		GradleVersion:       "9.7.1", GradleArchiveSHA256: "acd53f1edaf02f1a8ff99879f8a34b302661a057d9b063ae9e35b552f804d20a",
		JDKVersion: "21.0.12+8", JDKArchiveSHA256: "e4446ff06a276155697597cc0f1b15da004ff083f4964a35271ecee567177370",
	}
}

type staticAudit struct {
	Schema                    string        `json:"schemaVersion"`
	State                     string        `json:"state"`
	ProtocolSHA256            string        `json:"protocolSha256"`
	Files                     []fileBinding `json:"files"`
	SourceTrees               []string      `json:"sourceTrees"`
	Package                   fileBinding   `json:"package"`
	PublicExecutionAuthorized bool          `json:"publicExecutionAuthorized"`
}

// auditStatic rechecks retained immutable source and commit responses offline.
// Hashing the JDK archive proves its downloaded bytes, not daemon/compiler
// selection or an installed runtime. Those identities require native evidence.
func auditStatic(cache, pkg, jdkArchive string) (staticAudit, error) {
	p := makeProtocol()
	raw, err := json.Marshal(p)
	if err != nil {
		return staticAudit{}, err
	}
	a := staticAudit{Schema: "buildopt.eic/static-input-audit/v1", State: "STATIC_BYTES_ONLY", ProtocolSHA256: digest(raw), Files: []fileBinding{}, SourceTrees: []string{}}
	inputs := sourceInputs()
	expected := []fileBinding{
		{Path: filepath.Join(cache, inputs.TaskPreimageSHA256), SHA256: inputs.TaskPreimageSHA256},
		{Path: filepath.Join(cache, inputs.OwnerInputSHA256), SHA256: inputs.OwnerInputSHA256},
		{Path: filepath.Join(cache, "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"), SHA256: "a5a5c199ba02189ae8c46a334223371a20599d9c298ef65e7540ede4a3f72d59"},
		{Path: filepath.Join(cache, "7a9ce74cff467ca1bf60a4fcd9f05185acceda4d0f382434d393e17864262c5d"), SHA256: "7a9ce74cff467ca1bf60a4fcd9f05185acceda4d0f382434d393e17864262c5d"},
		{Path: jdkArchive, SHA256: inputs.JDKArchiveSHA256},
	}
	responses := []string{
		"ba46d1a013a50a1bc01f04cca66705f5b35589955af14f75d7c10f48f090570e",
		"1dd5fd7a165d2e9c24e949b14989e65cbc849f9427393ad580ada082b3e3f6c2",
		"4ed04ced43fb94c2c86ccf419fffca3456131b24407d87048572d1e7355d7262",
		"2fd2847e9bc975a7415d68f2e89602d329817d2b60c26a245e53026e15d04d8d",
		"3aea0deb62519ed68919829b608b9f4ba501b6237ff19f4da9113a3b4b1a1c5a",
		"0244881c3759949dd0f8d9c9609ecef10c812064c844a9d6870604750f71de34",
	}
	for i, revision := range p.Revisions {
		expected = append(expected, fileBinding{Path: filepath.Join(cache, revision+".commit.json"), SHA256: responses[i]})
	}
	for _, binding := range expected {
		if err = checkBinding(binding); err != nil {
			return a, err
		}
		a.Files = append(a.Files, binding)
	}
	parent := "f5064f49f2c4016159f07adf9e5eaff0a0a9c4a1"
	for _, revision := range p.Revisions {
		// Raw response hashes above are frozen. Decode only upstream fields needed
		// for ancestry; unknown upstream fields remain covered by the raw digest.
		var c struct {
			SHA     string `json:"sha"`
			Parents []struct {
				SHA string `json:"sha"`
			} `json:"parents"`
			Commit struct {
				Tree struct {
					SHA string `json:"sha"`
				} `json:"tree"`
			} `json:"commit"`
		}
		data, e := os.ReadFile(filepath.Join(cache, revision+".commit.json"))
		if e != nil {
			return a, e
		}
		if e = json.Unmarshal(data, &c); e != nil {
			return a, e
		}
		if c.SHA != revision || len(c.Parents) == 0 || c.Parents[0].SHA != parent || len(c.Commit.Tree.SHA) != 40 {
			return a, errors.New("frozen first-parent ancestry mismatch")
		}
		a.SourceTrees = append(a.SourceTrees, c.Commit.Tree.SHA)
		parent = revision
	}
	a.Package.Path = pkg
	a.Package.SHA256, err = hashFile(pkg)
	if err != nil {
		return a, err
	}
	if err = checkBinding(a.Package); err != nil {
		return a, err
	}
	return a, nil
}
