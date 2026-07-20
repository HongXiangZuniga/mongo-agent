// Package mongodb implementa el adapter de salida (driven) hacia MongoDB.
//
// Este archivo contiene VerifyReadOnlyGuarantee, la verificación activa de
// arranque que garantiza que el usuario de MongoDB no puede escribir.
package mongodb

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/HongXiangZuniga/agente-inaricards/pkg/utils"
)

// VerifyReadOnlyGuarantee intenta insertar un documento de prueba en una
// colección temporal. Si la inserción tiene éxito, el usuario puede escribir
// y se devuelve un error. Si falla por falta de autorización, la garantía de
// solo lectura se confirma y se devuelve nil.
func VerifyReadOnlyGuarantee(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("__readonly_check__")
	doc := bson.M{"canary": true, "ts": time.Now()}

	res, err := collection.InsertOne(ctx, doc)
	if err != nil {
		if isAuthorizationError(err) {
			return nil
		}
		return err
	}

	// Best-effort: limpiar el documento de prueba si la inserción tuvo éxito.
	_, _ = collection.DeleteOne(ctx, bson.M{"_id": res.InsertedID})
	return utils.ErrMongoUserNotReadOnly()
}

// isAuthorizationError reporta si err es un fallo de autorización de MongoDB
// (código 13, Unauthorized). Cubre las distintas formas que el driver puede
// devolver para un insert rechazado (CommandError o WriteException) y los
// mensajes tanto de MongoDB self-hosted ("not authorized on ...") como de
// Atlas ("user is not allowed to do action [insert] on [db.coll]").
func isAuthorizationError(err error) bool {
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Code == 13 {
		return true
	}

	var writeEx mongo.WriteException
	if errors.As(err, &writeEx) {
		for _, we := range writeEx.WriteErrors {
			if we.Code == 13 {
				return true
			}
		}
		if writeEx.WriteConcernError != nil && writeEx.WriteConcernError.Code == 13 {
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not authorized") ||
		strings.Contains(msg, "not allowed to do action")
}
