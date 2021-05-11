package version

import "testing"

func TestVersion(t *testing.T) {
	// Check the default version.
	v := Version()
	if v != "devel" {
		t.Errorf("expected default version 'devel', got '%s'", v)
	}

	// Now set values
	version = "testversion"
	commitHash = "testhash"
	v = Version()
	if v != "testversion-testhash" {
		t.Errorf("expected version 'testversion-testhash', got '%s'", v)
	}
}

func TestBuildDate(t *testing.T) {
	// Check the default build date
	if b := BuildDate(); b != "unknown" {
		t.Errorf("expected build date 'unknown', got '%s'", b)
	}

	// Set build date
	buildDate = "setdate"
	if b := BuildDate(); b != buildDate {
		t.Errorf("expected build date 'setdate', got '%s'", b)
	}
}
