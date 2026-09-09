package git

import (
	"testing"
)

func TestParseUnifiedDiff(t *testing.T) {
	diff := `diff --git a/pkg/auth/login.go b/pkg/auth/login.go
index 1234567..89abcdef 100644
--- a/pkg/auth/login.go
+++ b/pkg/auth/login.go
@@ -10,6 +10,8 @@ package auth
 import "fmt"
 
 func Login(user, pass string) bool {
+	// Added authentication check
+	fmt.Println("Authenticating user...")
 	return user == "admin"
 }
diff --git a/pkg/db/query.go b/pkg/db/query.go
new file mode 100644
--- /dev/null
+++ b/pkg/db/query.go
@@ -0,0 +1,5 @@
+package db
+
+func Query() {
+	// new file
+}
`

	files := ParseUnifiedDiff(diff)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Check file 1
	if files[0].OldPath != "pkg/auth/login.go" || files[0].NewPath != "pkg/auth/login.go" {
		t.Errorf("file 0 path mismatch: old=%s, new=%s", files[0].OldPath, files[0].NewPath)
	}
	if len(files[0].ChangedLines) != 2 {
		t.Errorf("expected 2 changed lines, got %d (%v)", len(files[0].ChangedLines), files[0].ChangedLines)
	}
	if files[0].ChangedLines[0] != 13 || files[0].ChangedLines[1] != 14 {
		t.Errorf("unexpected changed lines: %v", files[0].ChangedLines)
	}

	// Check file 2
	if !files[1].IsNew {
		t.Errorf("expected file 2 to be new")
	}
	if files[1].NewPath != "pkg/db/query.go" {
		t.Errorf("unexpected path for file 2: %s", files[1].NewPath)
	}
	if len(files[1].ChangedLines) != 5 {
		t.Errorf("expected 5 changed lines for new file, got %d", len(files[1].ChangedLines))
	}
}

func TestLineInDiff(t *testing.T) {
	changed := []int{10, 11, 12, 45, 50}

	if !IsLineInDiff(11, changed) {
		t.Errorf("expected line 11 to be in diff")
	}
	if IsLineInDiff(15, changed) {
		t.Errorf("expected line 15 not to be in diff")
	}

	nearest := FindNearestDiffLine(16, changed)
	if nearest != 12 {
		t.Errorf("expected nearest line to 16 to be 12, got %d", nearest)
	}
}
