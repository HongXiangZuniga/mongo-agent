package mongodb

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

// TestIsAuthorizationError cubre las distintas formas en que el driver de
// MongoDB puede reportar un insert rechazado por falta de privilegios:
// CommandError con código 13, WriteException con código 13, y los mensajes
// de texto de MongoDB self-hosted ("not authorized") y de Atlas
// ("user is not allowed to do action [...]").
func TestIsAuthorizationError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "command error code 13",
			err:  mongo.CommandError{Code: 13, Message: "not authorized on db.coll to execute command"},
			want: true,
		},
		{
			name: "write exception with code 13",
			err: mongo.WriteException{
				WriteErrors: []mongo.WriteError{
					{Index: 0, Code: 13, Message: "user is not allowed to do action [insert] on [db.coll]"},
				},
			},
			want: true,
		},
		{
			name: "write concern error with code 13",
			err: mongo.WriteException{
				WriteConcernError: &mongo.WriteConcernError{Code: 13, Message: "not authorized"},
			},
			want: true,
		},
		{
			name: "atlas style message",
			err:  errors.New("user is not allowed to do action [insert] on [inari_cards.carousel]"),
			want: true,
		},
		{
			name: "self-hosted style message",
			err:  errors.New("not authorized on db.coll to execute command { insert: \"coll\" }"),
			want: true,
		},
		{
			name: "network error is not an authorization error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "duplicate key error is not an authorization error",
			err: mongo.WriteException{
				WriteErrors: []mongo.WriteError{
					{Index: 0, Code: 11000, Message: "E11000 duplicate key error"},
				},
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAuthorizationError(tc.err); got != tc.want {
				t.Fatalf("isAuthorizationError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
