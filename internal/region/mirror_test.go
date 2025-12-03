package region

import "testing"

func TestSelectMirror(t *testing.T) {
	t.Parallel()

	cases := []struct {
		country string
		want    MirrorConfig
	}{
		{country: "CN", want: StudyGolangMirror},
		{country: "cn", want: StudyGolangMirror},
		{country: "  cn  ", want: StudyGolangMirror},
		{country: "US", want: GoDevMirror},
		{country: "", want: GoDevMirror},
	}

	for _, tc := range cases {
		got := SelectMirror(tc.country)
		if got != tc.want {
			t.Fatalf("SelectMirror(%q)=%v want %v", tc.country, got, tc.want)
		}
	}
}
