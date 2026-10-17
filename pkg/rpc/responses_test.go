package rpc

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEpochCredit_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    EpochCredit
		wantErr bool
	}{
		{"typical", `[1060, 2717157121, 2715278265]`, EpochCredit{1060, 2717157121, 2715278265}, false},
		{
			// agave appends an all-u64::MAX AG_MIGRATION_EPOCH_CREDIT entry when a vote account migrates to Alpenglow:
			"alpenglow migration marker",
			`[18446744073709551615, 18446744073709551615, 18446744073709551615]`,
			EpochCredit{math.MaxUint64, math.MaxUint64, math.MaxUint64},
			false,
		},
		{"too short", `[1060, 10]`, EpochCredit{}, true},
		{"too long", `[1060, 10, 0, 0]`, EpochCredit{}, true},
		{"not an array", `{"epoch": 1060}`, EpochCredit{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got EpochCredit
			err := json.Unmarshal([]byte(tt.in), &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEpochCredit_MarshalJSON_roundTrip(t *testing.T) {
	in := []EpochCredit{{Epoch: 1, Credits: 64, PreviousCredits: 0}, {Epoch: 2, Credits: 192, PreviousCredits: 64}}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	assert.JSONEq(t, `[[1,64,0],[2,192,64]]`, string(data))

	var out []EpochCredit
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in, out)
}

func TestVoteAccount_CreditsInEpoch(t *testing.T) {
	history := []EpochCredit{
		{Epoch: 1058, Credits: 2708382459, PreviousCredits: 2701473179},
		{Epoch: 1059, Credits: 2715278265, PreviousCredits: 2708382459},
		{Epoch: 1060, Credits: 2717157121, PreviousCredits: 2715278265},
	}
	tests := []struct {
		name    string
		account VoteAccount
		epoch   int64
		want    uint64
	}{
		{"current epoch", VoteAccount{EpochCredits: history}, 1060, 1878856},
		{"completed epoch", VoteAccount{EpochCredits: history}, 1059, 6895806},
		{"epoch not in history (delinquent since before the epoch)", VoteAccount{EpochCredits: history}, 1061, 0},
		{"empty history", VoteAccount{}, 1060, 0},
		{"negative epoch", VoteAccount{EpochCredits: history}, -1, 0},
		{
			"alpenglow migration marker is not matched",
			VoteAccount{EpochCredits: []EpochCredit{
				{Epoch: 1060, Credits: 100, PreviousCredits: 0},
				{Epoch: math.MaxUint64, Credits: math.MaxUint64, PreviousCredits: math.MaxUint64},
			}},
			1061, 0,
		},
		{
			"credits below previousCredits does not underflow",
			VoteAccount{EpochCredits: []EpochCredit{{Epoch: 1060, Credits: 5, PreviousCredits: 10}}},
			1060, 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.account.CreditsInEpoch(tt.epoch))
		})
	}
}
