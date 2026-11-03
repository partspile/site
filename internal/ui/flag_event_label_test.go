package ui

import "testing"

func TestFlagEventLabel(t *testing.T) {
	base := FlagEventData{
		OwnerID:      1,
		InquirerID:   2,
		OwnerName:    "alice",
		InquirerName: "bob",
	}

	tests := []struct {
		name string
		d    FlagEventData
		want string
	}{
		{
			name: "inquirer threw, self view",
			d: FlagEventData{
				ThrowerID: 2, CurrentUserID: 2,
				Kind:    FlagEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at ad by you",
		},
		{
			name: "inquirer unthrew, self view",
			d: FlagEventData{
				ThrowerID: 2, CurrentUserID: 2,
				Kind:    FlagEventLowern,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag lowered at ad by you",
		},
		{
			name: "inquirer threw, owner view",
			d: FlagEventData{
				ThrowerID: 2, CurrentUserID: 1,
				Kind:    FlagEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at ad by bob",
		},
		{
			name: "owner threw, inquirer view",
			d: FlagEventData{
				ThrowerID: 1, CurrentUserID: 2,
				Kind:    FlagEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at you by alice",
		},
		{
			name: "owner threw, self view",
			d: FlagEventData{
				ThrowerID: 1, CurrentUserID: 1,
				Kind:    FlagEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at bob by you",
		},
		{
			name: "owner threw, public view",
			d: FlagEventData{
				ThrowerID: 1, CurrentUserID: 99,
				Kind:    FlagEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at bob by alice",
		},
		{
			name: "inquirer threw with policy reason",
			d: FlagEventData{
				ThrowerID: 2, CurrentUserID: 2,
				Kind:    FlagEventThrown,
				Reason:  "policy",
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at ad by you · Listing or content violates policies",
		},
		{
			name: "owner threw with deal reason",
			d: FlagEventData{
				ThrowerID: 1, CurrentUserID: 1,
				Kind:    FlagEventThrown,
				Reason:  "deal",
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at bob by you · Deal or meetup went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flagEventLabel(tt.d)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
