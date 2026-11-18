package utils

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetItem[T any](ctx context.Context, client *firestore.Client, collection string, document string, field string) (T, error) {
	var err error
	var result T
	var snapshot *firestore.DocumentSnapshot
	var fieldValue any
	var ErrWrongType = errors.New("found field has wrong type")

	doc := client.Collection(collection).Doc(document)

	snapshot, err = doc.Get(ctx)
	if err != nil {
		log.Printf("fetching cursor document from firestore: %v", err)
		return result, err
	}

	fieldValue, err = snapshot.DataAt(field)
	if err != nil {
		log.Printf("reading document field: %v", err)
		return result, err
	}
	result, ok := fieldValue.(T)
	if ok {
		return result, nil
	}

	encoded, err := json.Marshal(fieldValue)
	if err == nil {
		err = json.Unmarshal(encoded, &result)
	}
	if err != nil {
		var zero T
		log.Printf("%s is not a %T, got %T", field, zero, fieldValue)
		return zero, ErrWrongType
	}

	return result, nil
}

func IsNotFound(err error) bool {
	var noField *firestore.FieldNotFoundError
	return status.Code(err) == codes.NotFound || errors.As(err, &noField)
}

func SetItem(ctx context.Context, client *firestore.Client, collection string, document string, field string, value any) error {
	doc := client.Collection(collection).Doc(document)

	_, err := doc.Set(ctx, map[string]any{field: value}, firestore.MergeAll)
	if err != nil {
		log.Printf("writing %s.%s field %q: %v", collection, document, field, err)
		return err
	}

	return nil
}
