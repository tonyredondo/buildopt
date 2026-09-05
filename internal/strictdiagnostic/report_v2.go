package strictdiagnostic

// SelectRootReportV2 treats repeated, textually identical report references as
// one logical identity. Gradle repeats the same failure summary in stack traces;
// distinct references remain ambiguous and inventory is still not authority.
func SelectRootReportV2(log []byte, expectedRoot string) Selection {
	return selectRootReport(log, expectedRoot, true)
}
