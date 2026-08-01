//go:build linux

package sharedcache

import "testing"

func TestFilesystemTypePolicyAcceptsOnlyProvenLocalFilesystems(t *testing.T) {
	for _, filesystemType := range []int64{
		bcachefsMagic, btrfsSuperMagic, ecryptfsMagic, extSuperMagic,
		f2fsSuperMagic, nilfsSuperMagic, overlayfsMagic, ramfsMagic,
		reiserfsMagic, tmpfsMagic, ubifsMagic, xfsSuperMagic, zfsSuperMagic,
	} {
		if !isSupportedLocalFilesystemType(filesystemType) {
			t.Errorf("local filesystem type %#x was rejected", filesystemType)
		}
	}
	for _, filesystemType := range []int64{
		afsSuperMagic, cephSuperMagic, cifsSuperMagic, codaSuperMagic,
		fuseSuperMagic, gfs2SuperMagic, ncpSuperMagic, nfsSuperMagic,
		ocfs2SuperMagic, smbSuperMagic, v9fsSuperMagic, 0x12345678,
	} {
		if isSupportedLocalFilesystemType(filesystemType) {
			t.Errorf("filesystem type %#x was accepted", filesystemType)
		}
	}
}
