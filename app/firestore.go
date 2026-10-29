package function

import (
	"context"
	"errors"
	"log"

	"cloud.google.com/go/firestore"
)

func getFirestoreItem[T any](ctx context.Context, collection string, document string, field string) (T, error) {
	var err error
	var result T
	var snapshot *firestore.DocumentSnapshot
	var fieldValue any
	var ErrWrongType = errors.New("found field has wrong type")

	doc := firestoreClient.Collection(collection).Doc(document)

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
	if !ok {
		log.Printf("historyId is not a %T, got %T", result, fieldValue)
		return result, ErrWrongType
	}

	return result, nil
}

func setFirestoreItem(ctx context.Context, collection string, document string, field string, value any) error {
	doc := firestoreClient.Collection(collection).Doc(document)

	_, err := doc.Set(ctx, map[string]any{field: value}, firestore.MergeAll)
	if err != nil {
		log.Printf("writing %s.%s field %q: %v", collection, document, field, err)
		return err
	}

	return nil
}
