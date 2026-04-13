package workers

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestUuidToStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		uuid pgtype.UUID
		want string
	}{
		{
			name: "valid uuid",
			uuid: pgtype.UUID{
				Bytes: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
				Valid: true,
			},
			want: "01020304-0506-0708-090a-0b0c0d0e0f10",
		},
		{
			name: "zero uuid",
			uuid: pgtype.UUID{
				Bytes: [16]byte{},
				Valid: true,
			},
			want: "00000000-0000-0000-0000-000000000000",
		},
		{
			name: "invalid uuid",
			uuid: pgtype.UUID{Valid: false},
			want: "",
		},
		{
			name: "all ff",
			uuid: pgtype.UUID{
				Bytes: [16]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				Valid: true,
			},
			want: "ffffffff-ffff-ffff-ffff-ffffffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := uuidToStr(tt.uuid)
			if got != tt.want {
				t.Errorf("uuidToStr() = %q (len %d), want %q (len %d)", got, len(got), tt.want, len(tt.want))
			}
			if tt.uuid.Valid && len(got) != 36 {
				t.Errorf("uuidToStr() length = %d, want 36", len(got))
			}
		})
	}
}
