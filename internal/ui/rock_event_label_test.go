package ui

import "testing"

func TestRockEventLabel(t *testing.T) {
	base := RockEventData{
		OwnerID:      1,
		InquirerID:   2,
		OwnerName:    "sfeldma",
		InquirerName: "bob",
	}

	tests := []struct {
		name string
		d    RockEventData
		want string
	}{
		{
			name: "inquirer threw, self view",
			d: RockEventData{
				ThrowerID: 2, CurrentUserID: 2,
				Kind:    RockEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at ad by you",
		},
		{
			name: "inquirer unthrew, self view",
			d: RockEventData{
				ThrowerID: 2, CurrentUserID: 2,
				Kind:    RockEventUnthrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag lowered at ad by you",
		},
		{
			name: "inquirer threw, owner view",
			d: RockEventData{
				ThrowerID: 2, CurrentUserID: 1,
				Kind:    RockEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at ad by bob",
		},
		{
			name: "owner threw, inquirer view",
			d: RockEventData{
				ThrowerID: 1, CurrentUserID: 2,
				Kind:    RockEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at you by sfeldma",
		},
		{
			name: "owner threw, self view",
			d: RockEventData{
				ThrowerID: 1, CurrentUserID: 1,
				Kind:    RockEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at bob by you",
		},
		{
			name: "owner threw, public view",
			d: RockEventData{
				ThrowerID: 1, CurrentUserID: 99,
				Kind:    RockEventThrown,
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at bob by sfeldma",
		},
		{
			name: "inquirer threw with policy reason",
			d: RockEventData{
				ThrowerID: 2, CurrentUserID: 2,
				Kind:    RockEventThrown,
				Reason:  "policy",
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at ad by you · Listing or content violates policies",
		},
		{
			name: "owner threw with deal reason",
			d: RockEventData{
				ThrowerID: 1, CurrentUserID: 1,
				Kind:    RockEventThrown,
				Reason:  "deal",
				OwnerID: base.OwnerID, InquirerID: base.InquirerID,
				OwnerName: base.OwnerName, InquirerName: base.InquirerName,
			},
			want: "Flag raised at bob by you · Deal or meetup went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rockEventLabel(tt.d)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
