package spanner

import (
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
)

func TestInformationSchemaColumnOption_isImplicitlyAssigned(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		option   *informationSchemaColumnOption
		expected bool
	}{
		{
			// Cloud Spanner exposes this for every column even when the user has never
			// specified a locality group. Including it makes ddlctl emit an invalid
			// `ALTER COLUMN ... DROP OPTIONS` against DDL that has no OPTIONS clause.
			name:     "true,locality_group=default is implicitly assigned",
			option:   &informationSchemaColumnOption{ColumnName: "Id", OptionName: "locality_group", OptionValue: "default"},
			expected: true,
		},
		{
			// An explicitly specified locality group must still be reported so that
			// changes to it are detected.
			name:     "false,locality_group with a non-default value is explicit",
			option:   &informationSchemaColumnOption{ColumnName: "Id", OptionName: "locality_group", OptionValue: "archive"},
			expected: false,
		},
		{
			name:     "false,allow_commit_timestamp is explicit",
			option:   &informationSchemaColumnOption{ColumnName: "CreatedAt", OptionName: "allow_commit_timestamp", OptionValue: "TRUE"},
			expected: false,
		},
		{
			// Guard against matching on the value alone.
			name:     "false,another option whose value happens to be default",
			option:   &informationSchemaColumnOption{ColumnName: "Id", OptionName: "some_other_option", OptionValue: "default"},
			expected: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.option.isImplicitlyAssigned())
		})
	}
}
