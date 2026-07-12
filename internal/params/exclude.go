package params

import "regexp"

type excludeRule struct {
	re     *regexp.Regexp
	score  int
	reason string
}

var excludeRules = []excludeRule{
	{regexp.MustCompile(`(?i)^utm_`), 12, "analytics UTM parameter"},
	{regexp.MustCompile(`(?i)^(gclid|fbclid|msclkid|twclid|li_fat_id|igshid|yclid|wbraid|gbraid|dclid|srsltid)$`), 10, "ad click tracking parameter"},
	{regexp.MustCompile(`(?i)^(_ga|_gid|_gat|_hj|_hsq|_hp2|_fbp|_fbc|uet|mc_eid|mc_cid|pk_campaign|pk_kwd)$`), 15, "analytics identifier"},
	{regexp.MustCompile(`(?i)^(csrf|xsrf|nonce|_nonce)$`), 18, "CSRF/nonce token"},
	{regexp.MustCompile(`(?i)^(phpsessid|jsessionid|asp\.net_sessionid|sid|session_id|sessionid|sess|auth_token|access_token|refresh_token|bearer)$`), 20, "session/auth token"},
	{regexp.MustCompile(`(?i)^(width|height|color|colour|theme|skin|font|fontsize|bg|background|foreground|opacity|margin|padding|border|align|valign)$`), 22, "UI/layout parameter"},
	{regexp.MustCompile(`(?i)^(lang|locale|language|hl|region|country|currency|timezone|tz|format)$`), 25, "locale/display parameter"},
	{regexp.MustCompile(`(?i)^(referrer|referer|ref_src|source|medium|campaign|content|term)$`), 20, "referral/marketing parameter"},
	{regexp.MustCompile(`(?i)^(cache|nocache|timestamp|ts|rand|random|_t|cb|cachebuster|ver|version|rev|build|v)$`), 28, "cache-busting parameter"},
	{regexp.MustCompile(`(?i)^(__viewstate|__eventvalidation|__requestdigest|__anti|viewstate|eventvalidation)$`), 15, "CMS viewstate parameter"},
	{regexp.MustCompile(`(?i)^(wpnonce|_wpnonce|action_nonce|nc|hash|sig|signature|checksum|token)$`), 30, "CMS/form security token"},
	{regexp.MustCompile(`(?i)^(redirect|redirect_uri|return|returnurl|return_url|next|continue|callback|cb_url|url_redirect)$`), 35, "redirect handler (open redirect risk, not SQLi)"},
	{regexp.MustCompile(`(?i)^(ajax|jsonp|callback|format|output|response_type|dataType)$`), 32, "response format parameter"},
	{regexp.MustCompile(`(?i)^(img|image|icon|logo|avatar|thumb|thumbnail|banner|photo|pic|src_img)$`), 30, "media asset parameter"},
	{regexp.MustCompile(`(?i)^(css|js|script|style|font|woff|asset|bundle|chunk)$`), 28, "static asset parameter"},
	{regexp.MustCompile(`(?i)^(sidebar|sidebars|widget|widgets|menu_id|nav_id|block_id|module_id|plugin|plugins|theme_mod|customize_messenger_channel)$`), 18, "CMS/layout parameter"},
	{regexp.MustCompile(`(?i)^(fb|twitter|linkedin|share|social|utm|tracking|track|pixel|beacon)$`), 18, "social/tracking parameter"},
}

func matchExclude(name string) (int, string, bool) {
	for _, rule := range excludeRules {
		if rule.re.MatchString(name) {
			return rule.score, rule.reason, true
		}
	}
	return 0, "", false
}
