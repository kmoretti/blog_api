package friendsRepositories

import (
	"testing"

	"blog_api/src/model"
	"blog_api/src/testutil"
)

func TestUpdateFriendLinkByID_SerializesTagsAsJSON(t *testing.T) {
	db := testutil.NewTestDB(t)
	link := model.FriendWebsite{Name: "A", Link: "https://a.test"}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}

	req := model.EditFriendLinkReq{
		Data: map[string]interface{}{
			"tags": []interface{}{"blog", "test"},
		},
	}

	rows, err := UpdateFriendLinkByID(db, uint(link.ID), req)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	updated, err := GetFriendLinkByID(db, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Tags) != 2 || updated.Tags[0] != "blog" || updated.Tags[1] != "test" {
		t.Fatalf("expected tags [blog test], got %v", updated.Tags)
	}
}

func TestStringSliceToleratesInvalidJSON(t *testing.T) {
	db := testutil.NewTestDB(t)
	link := model.FriendWebsite{Name: "B", Link: "https://b.test"}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Model(&model.FriendWebsite{}).Where("id = ?", link.ID).Update("tags", "[not valid json").Error; err != nil {
		t.Fatal(err)
	}

	var fetched model.FriendWebsite
	if err := db.First(&fetched, link.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fetched.Tags == nil {
		t.Fatal("expected non-nil tags fallback")
	}
	if len(fetched.Tags) != 0 {
		t.Fatalf("expected empty tags, got %v", fetched.Tags)
	}
}
