package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals
var date = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

func TestIssue_FullPath(t *testing.T) {
	type fields struct {
		Date time.Time
		Ext  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"pdf", fields{Date: date, Ext: ".pdf"}, "2025/01/02.pdf"},
		{"no ext", fields{Date: date}, "2025/01/02"},
		{"no date", fields{Ext: ".pdf"}, "0001/01/01.pdf"},
		{"no date ext", fields{}, "0001/01/01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Issue{
				Date: tt.fields.Date,
				Ext:  tt.fields.Ext,
			}
			assert.Equal(t, tt.want, i.FullPath())
		})
	}
}

func TestIssue_ShortPath(t *testing.T) {
	type fields struct {
		Date time.Time
		Ext  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"pdf", fields{Date: date, Ext: ".pdf"}, "2025-01-02.pdf"},
		{"no ext", fields{Date: date}, "2025-01-02"},
		{"no date", fields{Ext: ".pdf"}, "0001-01-01.pdf"},
		{"no date ext", fields{}, "0001-01-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Issue{
				Date: tt.fields.Date,
				Ext:  tt.fields.Ext,
			}
			assert.Equal(t, tt.want, i.ShortPath())
		})
	}
}

func TestNewIssueFromDate(t *testing.T) {
	type args struct {
		date time.Time
		ext  string
	}
	tests := []struct {
		name string
		args args
		want *Issue
	}{
		{"basic", args{date: date, ext: ".pdf"}, &Issue{Date: date, Ext: ".pdf"}},
		{"no ext", args{date: date, ext: ""}, &Issue{Date: date, Ext: ""}},
		{"no date", args{ext: ".pdf"}, &Issue{Ext: ".pdf"}},
		{"no date ext", args{}, &Issue{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewIssueFromDate(tt.args.date, tt.args.ext))
		})
	}
}

func TestNewIssueFromPath(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name    string
		args    args
		want    *Issue
		wantErr require.ErrorAssertionFunc
	}{
		{"short", args{"2025-01-02.pdf"}, &Issue{Date: date, Ext: ".pdf"}, require.NoError},
		{"short invalid date", args{"2025-01-02abc.pdf"}, nil, require.Error},
		{"long", args{"test-issue-1-2-2025.pdf"}, &Issue{Date: date, Ext: ".pdf"}, require.NoError},
		{"long missing section", args{"test-1-2-2025.pdf"}, nil, require.Error},
		{"long invalid date", args{"test-issue-1-2-2025abc.pdf"}, nil, require.Error},
		{"empty", args{""}, nil, require.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewIssueFromPath(tt.args.p)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_getDate(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr require.ErrorAssertionFunc
	}{
		{"short", args{"2025-01-02.pdf"}, date, require.NoError},
		{"short invalid date", args{"2025-01-02abc.pdf"}, time.Time{}, require.Error},
		{"long", args{"test-issue-1-2-2025.pdf"}, date, require.NoError},
		{"long missing section", args{"test-1-2-2025.pdf"}, time.Time{}, require.Error},
		{"long invalid date", args{"test-issue-1-2-2025abc.pdf"}, time.Time{}, require.Error},
		{"empty", args{""}, time.Time{}, require.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDate(tt.args.filename)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
