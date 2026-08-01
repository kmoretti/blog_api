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

func TestUpdateFriendLinkByID_PreservesExistingValuesAndUpdatesRssFields(t *testing.T) {
	db := testutil.NewTestDB(t)
	link := model.FriendWebsite{
		Name:            "Original",
		Link:            "https://original.test",
		Avatar:          "https://original.test/avatar.png",
		Rss:             "https://original.test/rss.xml",
		RejectionReason: "old reason",
	}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := UpdateFriendLinkByID(db, uint(link.ID), model.EditFriendLinkReq{Data: map[string]interface{}{
		"rss":              "https://original.test/new-rss.xml",
		"rejection_reason": "new reason",
		"description":      "",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row affected, got %d", rows)
	}

	updated, err := GetFriendLinkByID(db, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Rss != "https://original.test/new-rss.xml" || updated.RejectionReason != "new reason" {
		t.Fatalf("expected RSS and rejection reason to update, got %#v", updated)
	}
	if updated.Name != link.Name || updated.Avatar != link.Avatar {
		t.Fatalf("expected omitted values to be preserved, got %#v", updated)
	}
}

func TestSetFriendLinkGroupsValidatesBeforeReplacingMappings(t *testing.T) {
	db := testutil.NewTestDB(t)
	link := model.FriendWebsite{Name: "A", Link: "https://a.test"}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	validGroup := model.FriendLinkGroup{Name: "valid"}
	if err := db.Create(&validGroup).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FriendLinkGroupMapping{FriendLinkID: link.ID, FriendLinkGroupID: validGroup.ID}).Error; err != nil {
		t.Fatal(err)
	}

	err := SetFriendLinkGroups(db, link.ID, []int{validGroup.ID, 99999})
	if err != ErrFriendLinkGroupNotFound {
		t.Fatalf("expected missing group error, got %v", err)
	}
	ids, err := GetFriendLinkGroupIDs(db, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != validGroup.ID {
		t.Fatalf("expected existing mapping to remain, got %v", ids)
	}
}

func TestUpdateAndDeleteFriendLinkGroupReturnNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	if err := UpdateFriendLinkGroup(db, 99999, &model.FriendLinkGroup{Name: "missing"}); err != ErrFriendLinkGroupNotFound {
		t.Fatalf("expected missing group update error, got %v", err)
	}
	if err := DeleteFriendLinkGroup(db, 99999); err != ErrFriendLinkGroupNotFound {
		t.Fatalf("expected missing group delete error, got %v", err)
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
