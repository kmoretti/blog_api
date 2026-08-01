package fcircle

import (
	"testing"

	"blog_api/src/model"
	"blog_api/src/testutil"
)

func TestBuildFriendJSONUsesDefaultGroupDescriptionForUngroupedLinks(t *testing.T) {
	db := testutil.NewTestDB(t)
	link := model.FriendWebsite{Name: "A", Link: "https://a.test"}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}

	data, err := buildFriendJSON(db, []model.FriendWebsite{link})
	if err != nil {
		t.Fatal(err)
	}
	groups, ok := data["linkGroups"].([]model.FriendLinkGroupOutput)
	if !ok || len(groups) != 1 {
		t.Fatalf("expected one output group, got %#v", data["linkGroups"])
	}
	if groups[0].Name != "网上邻居" || groups[0].Desc != "我的网上邻居~" {
		t.Fatalf("expected default group metadata, got %#v", groups[0])
	}
}
