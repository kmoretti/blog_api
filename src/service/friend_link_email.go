package service

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"fmt"
	"log"
	"strings"
)

// FriendLinkEmailData holds variables for friend link email templates.
type FriendLinkEmailData struct {
	SiteName         string
	SiteURL          string
	FriendName       string
	FriendLink       string
	FriendAvatar     string
	FriendDescription string
	FriendEmail      string
	RejectionReason  string
	AdminPanelURL    string
}

// NotifyFriendLinkApplication sends an email to the admin when a visitor
// submits a new friend link application.
func NotifyFriendLinkApplication(link model.FriendWebsite) {
	cfg := config.GetConfig()
	if !cfg.Email.Enable || cfg.Email.UserName == "" {
		log.Println("[email][friend] email disabled or admin address missing, skipping application notification")
		return
	}
	if !cfg.Email.FriendLinkAdminNotify {
		log.Println("[email][friend] admin notification disabled, skipping application notification")
		return
	}

	data := buildEmailData(cfg, link)
	content := EmailContent{
		Subject: fmt.Sprintf("[%s] 收到新的友链申请", data.SiteName),
		Body:    renderAdminApplicationTemplate(data),
		IsHTML:  true,
	}

	if err := SendEmail(cfg.Email, []string{cfg.Email.UserName}, content); err != nil {
		log.Printf("[email][friend] failed to send admin application notification: %v", err)
	} else {
		log.Println("[email][friend] admin application notification sent")
	}
}

// NotifyFriendLinkApproved sends an email to the applicant when their friend
// link application is approved.
func NotifyFriendLinkApproved(link model.FriendWebsite) {
	cfg := config.GetConfig()
	if !cfg.Email.Enable || link.Email == "" {
		log.Println("[email][friend] email disabled or applicant address missing, skipping approval notification")
		return
	}
	if !cfg.Email.FriendLinkUserNotify {
		log.Println("[email][friend] user notification disabled, skipping approval notification")
		return
	}

	data := buildEmailData(cfg, link)
	content := EmailContent{
		Subject: fmt.Sprintf("[%s] 友链申请已通过", data.SiteName),
		Body:    renderUserApprovedTemplate(data),
		IsHTML:  true,
	}

	if err := SendEmail(cfg.Email, []string{link.Email}, content); err != nil {
		log.Printf("[email][friend] failed to send approval notification: %v", err)
	} else {
		log.Println("[email][friend] approval notification sent")
	}
}

// NotifyFriendLinkRejected sends an email to the applicant when their friend
// link application is rejected, including the rejection reason if provided.
func NotifyFriendLinkRejected(link model.FriendWebsite) {
	cfg := config.GetConfig()
	if !cfg.Email.Enable || link.Email == "" {
		log.Println("[email][friend] email disabled or applicant address missing, skipping rejection notification")
		return
	}
	if !cfg.Email.FriendLinkUserNotify {
		log.Println("[email][friend] user notification disabled, skipping rejection notification")
		return
	}

	data := buildEmailData(cfg, link)
	content := EmailContent{
		Subject: fmt.Sprintf("[%s] 友链申请未通过", data.SiteName),
		Body:    renderUserRejectedTemplate(data),
		IsHTML:  true,
	}

	if err := SendEmail(cfg.Email, []string{link.Email}, content); err != nil {
		log.Printf("[email][friend] failed to send rejection notification: %v", err)
	} else {
		log.Println("[email][friend] rejection notification sent")
	}
}

func buildEmailData(cfg *model.Config, link model.FriendWebsite) FriendLinkEmailData {
	siteName := cfg.Site.Name
	if siteName == "" {
		siteName = "我的博客"
	}
	siteURL := cfg.Site.URL
	if siteURL == "" {
		siteURL = "https://example.com"
	}
	adminPanelURL := strings.TrimRight(siteURL, "/") + "/panel"

	return FriendLinkEmailData{
		SiteName:          siteName,
		SiteURL:           siteURL,
		FriendName:        link.Name,
		FriendLink:        link.Link,
		FriendAvatar:      link.Avatar,
		FriendDescription: link.Info,
		FriendEmail:       link.Email,
		RejectionReason:   link.RejectionReason,
		AdminPanelURL:     adminPanelURL,
	}
}

func renderAdminApplicationTemplate(data FriendLinkEmailData) string {
	tmpl := `<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); overflow: hidden; background-color: #ffffff;">
    <div style="background-color: #f8db8f; padding: 20px; color: white; text-align: center;">
      <div style="font-size: 28px; font-weight: bold;">👋 Hi 管理员</div>
      <div style="margin-top: 10px; font-size: 16px;">您在 <a href="${SITE_URL}" style="color: white; text-decoration: underline; font-weight: 500;">${SITE_NAME}</a> 收到一条新的友链申请！</div>
    </div>
    <div style="padding: 25px 20px;">
      <div style="margin-bottom: 20px;">
        <div style="font-weight: bold; margin-bottom: 10px; color: #333; font-size: 16px;">申请站点信息：</div>
        <div style="background-color: #f5f7fa; border-radius: 6px; padding: 15px; border: 1px solid #ebf0f5;">
          <div style="margin-bottom: 8px;"><strong>站点名称：</strong>${FRIEND_NAME}</div>
          <div style="margin-bottom: 8px;"><strong>站点地址：</strong><a href="${FRIEND_LINK}" style="color: #f8db8f; font-weight: 500;">${FRIEND_LINK}</a></div>
          <div style="margin-bottom: 8px;"><strong>联系邮箱：</strong>${FRIEND_EMAIL}</div>
          <div style="margin-bottom: 8px;"><strong>站点描述：</strong>${FRIEND_DESCRIPTION}</div>
        </div>
      </div>
      <div style="margin-bottom: 30px;">
        <a href="${ADMIN_PANEL_URL}" style="display: inline-block; background-color: #f8db8f; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; font-weight: 500;">前往审核</a>
      </div>
      <div style="color: #86909c; font-size: 12px; text-align: center; padding-top: 15px; border-top: 1px solid #ebf0f5;">此邮件由系统发送，请不要回复此邮件</div>
    </div>
  </div>
</div>`
	return substituteTemplateVars(tmpl, data)
}

func renderUserApprovedTemplate(data FriendLinkEmailData) string {
	tmpl := `<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); overflow: hidden; background-color: #ffffff;">
    <div style="background-color: #f8db8f; padding: 20px; color: white; text-align: center;">
      <div style="font-size: 28px; font-weight: bold;">🎉 恭喜您</div>
      <div style="margin-top: 10px; font-size: 16px;">您在 <a href="${SITE_URL}" style="color: white; text-decoration: underline; font-weight: 500;">${SITE_NAME}</a> 的友链申请已通过审核！</div>
    </div>
    <div style="padding: 25px 20px;">
      <div style="margin-bottom: 20px;">
        <div style="font-weight: bold; margin-bottom: 10px; color: #333; font-size: 16px;">您的站点信息：</div>
        <div style="background-color: #f5f7fa; border-radius: 6px; padding: 15px; border: 1px solid #ebf0f5;">
          <div style="margin-bottom: 8px;"><strong>站点名称：</strong>${FRIEND_NAME}</div>
          <div style="margin-bottom: 8px;"><strong>站点地址：</strong><a href="${FRIEND_LINK}" style="color: #f8db8f; font-weight: 500;">${FRIEND_LINK}</a></div>
          <div style="margin-bottom: 8px;"><strong>站点描述：</strong>${FRIEND_DESCRIPTION}</div>
        </div>
      </div>
      <div style="margin-bottom: 30px;">
        <a href="${SITE_URL}/friends" style="display: inline-block; background-color: #f8db8f; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; font-weight: 500;">查看友链页面</a>
      </div>
      <div style="color: #86909c; font-size: 12px; text-align: center; padding-top: 15px; border-top: 1px solid #ebf0f5;">此邮件由系统发送，请不要回复此邮件</div>
    </div>
  </div>
</div>`
	return substituteTemplateVars(tmpl, data)
}

func renderUserRejectedTemplate(data FriendLinkEmailData) string {
	reasonBlock := ""
	if strings.TrimSpace(data.RejectionReason) != "" {
		reasonBlock = `<div style="margin-bottom: 20px;">
        <div style="font-weight: bold; margin-bottom: 10px; color: #333; font-size: 16px;">拒绝原因：</div>
        <div style="background-color: #fff5f5; border-radius: 6px; padding: 15px; border: 1px solid #ffccc7; color: #cf1322;">${REJECTION_REASON}</div>
      </div>`
	}

	tmpl := `<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); overflow: hidden; background-color: #ffffff;">
    <div style="background-color: #f8db8f; padding: 20px; color: white; text-align: center;">
      <div style="font-size: 28px; font-weight: bold;">📧 很抱歉</div>
      <div style="margin-top: 10px; font-size: 16px;">您在 <a href="${SITE_URL}" style="color: white; text-decoration: underline; font-weight: 500;">${SITE_NAME}</a> 的友链申请未通过审核。</div>
    </div>
    <div style="padding: 25px 20px;">
      <div style="margin-bottom: 20px;">
        <div style="font-weight: bold; margin-bottom: 10px; color: #333; font-size: 16px;">您的站点信息：</div>
        <div style="background-color: #f5f7fa; border-radius: 6px; padding: 15px; border: 1px solid #ebf0f5;">
          <div style="margin-bottom: 8px;"><strong>站点名称：</strong>${FRIEND_NAME}</div>
          <div style="margin-bottom: 8px;"><strong>站点地址：</strong><a href="${FRIEND_LINK}" style="color: #f8db8f; font-weight: 500;">${FRIEND_LINK}</a></div>
        </div>
      </div>
      ` + reasonBlock + `
      <div style="margin-bottom: 30px;">
        <a href="${SITE_URL}/friends" style="display: inline-block; background-color: #f8db8f; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; font-weight: 500;">重新申请</a>
      </div>
      <div style="color: #86909c; font-size: 12px; text-align: center; padding-top: 15px; border-top: 1px solid #ebf0f5;">此邮件由系统发送，请不要回复此邮件</div>
    </div>
  </div>
</div>`
	return substituteTemplateVars(tmpl, data)
}

func substituteTemplateVars(tmpl string, data FriendLinkEmailData) string {
	replacements := map[string]string{
		"${SITE_NAME}":           data.SiteName,
		"${SITE_URL}":            data.SiteURL,
		"${FRIEND_NAME}":         data.FriendName,
		"${FRIEND_LINK}":         data.FriendLink,
		"${FRIEND_AVATAR}":       data.FriendAvatar,
		"${FRIEND_DESCRIPTION}":  data.FriendDescription,
		"${FRIEND_EMAIL}":        data.FriendEmail,
		"${REJECTION_REASON}":    data.RejectionReason,
		"${ADMIN_PANEL_URL}":     data.AdminPanelURL,
	}
	for key, val := range replacements {
		tmpl = strings.ReplaceAll(tmpl, key, val)
	}
	return tmpl
}
