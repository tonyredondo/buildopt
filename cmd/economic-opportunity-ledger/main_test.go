package main

import "testing"

func TestClassifyUsesPathsNotFamilyNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		paths      []string
		wantClass  string
		wantAction string
	}{
		{name: "owner only", paths: []string{"module/src/Main.java"}, wantClass: "OWNER_ONLY", wantAction: "ADMIT_NATIVE_CEILING_PROBE"},
		{name: "outside owner", paths: []string{"other/src/Main.java"}, wantClass: "OUTSIDE_OWNER", wantAction: "NO_ACTION"},
		{name: "mixed boundary", paths: []string{"module/src/Main.java", "other/src/Main.java"}, wantClass: "MIXED_OWNER_BOUNDARY", wantAction: "REJECT_INCOMPLETE_OR_AMBIGUOUS"},
		{name: "root build logic", paths: []string{"build.gradle.kts", "module/src/Main.java"}, wantClass: "GLOBAL_OR_BUILD_LOGIC", wantAction: "NO_ACTION"},
		{name: "empty", paths: nil, wantClass: "INCOMPLETE", wantAction: "REJECT_INCOMPLETE_OR_AMBIGUOUS"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			changeClass, decision, _ := classify(test.paths, []string{"module/"}, []string{"build.gradle.kts"})
			if changeClass != test.wantClass || decision != test.wantAction {
				t.Fatalf("classify() = %s/%s, want %s/%s", changeClass, decision, test.wantClass, test.wantAction)
			}
		})
	}
}
