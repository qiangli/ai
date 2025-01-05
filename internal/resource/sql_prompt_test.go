package resource

import (
	_ "embed"
	"testing"
)

func TestGetSqlSystemRoleContent(t *testing.T) {
	version := "PostgreSQL 15.8 on aarch64-unknown-linux-musl, compiled by gcc (Alpine 13.2.1_git20240309) 13.2.1 20240309, 64-bit"
	info := DBInfo{
		Version:     version,
		ContextData: "Not available",
	}
	msg, err := GetSqlSystemRoleContent(&info)
	if err != nil {
		t.Errorf("failed, expected nil, got %v", err)
	}
	t.Logf("\nSystem role content:\n%s\n", msg)
}
