package api

import (
	"fmt"
	"sort"
	"strings"

	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/setting"
)

const (
	// Themes
	lightName = "light"
	darkName  = "dark"
)

func (hs *HTTPServer) setIndexViewData(c *m.ReqContext) (*dtos.IndexViewData, error) {
	settings, err := hs.getFrontendSettingsMap(c)
	if err != nil {
		return nil, err
	}

	prefsQuery := m.GetPreferencesWithDefaultsQuery{User: c.SignedInUser}
	if err := bus.Dispatch(&prefsQuery); err != nil {
		return nil, err
	}
	prefs := prefsQuery.Result

	// Read locale from acccept-language
	acceptLang := c.Req.Header.Get("Accept-Language")
	locale := "en-US"

	if len(acceptLang) > 0 {
		parts := strings.Split(acceptLang, ",")
		locale = parts[0]
	}

	appURL := setting.AppUrl
	appSubURL := setting.AppSubUrl

	// special case when doing localhost call from phantomjs
	if c.IsRenderCall {
		appURL = fmt.Sprintf("%s://localhost:%s", setting.Protocol, setting.HttpPort)
		appSubURL = ""
		settings["appSubUrl"] = ""
	}

	hasEditPermissionInFoldersQuery := m.HasEditPermissionInFoldersQuery{SignedInUser: c.SignedInUser}
	if err := bus.Dispatch(&hasEditPermissionInFoldersQuery); err != nil {
		return nil, err
	}

	var data = dtos.IndexViewData{
		User: &dtos.CurrentUser{
			Id:                         c.UserId,
			IsSignedIn:                 c.IsSignedIn,
			Login:                      c.Login,
			Email:                      c.Email,
			Name:                       c.Name,
			OrgCount:                   c.OrgCount,
			OrgId:                      c.OrgId,
			OrgName:                    c.OrgName,
			OrgRole:                    c.OrgRole,
			GravatarUrl:                dtos.GetGravatarUrl(c.Email),
			IsGrafanaAdmin:             c.IsGrafanaAdmin,
			LightTheme:                 prefs.Theme == lightName,
			Timezone:                   prefs.Timezone,
			Locale:                     locale,
			HelpFlags1:                 c.HelpFlags1,
			HasEditPermissionInFolders: hasEditPermissionInFoldersQuery.Result,
		},
		Settings:                settings,
		Theme:                   prefs.Theme,
		AppUrl:                  appURL,
		AppSubUrl:               appSubURL,
		GoogleAnalyticsId:       setting.GoogleAnalyticsId,
		GoogleTagManagerId:      setting.GoogleTagManagerId,
		BuildVersion:            setting.BuildVersion,
		BuildCommit:             setting.BuildCommit,
		NewGrafanaVersion:       plugins.GrafanaLatestVersion,
		NewGrafanaVersionExists: plugins.GrafanaHasUpdate,
		AppName:                 setting.ApplicationName,
		AppNameBodyClass:        getAppNameBodyClass(hs.License.HasValidLicense()),
	}

	if setting.DisableGravatar {
		data.User.GravatarUrl = setting.AppSubUrl + "/public/img/user_profile.png"
	}

	if len(data.User.Name) == 0 {
		data.User.Name = data.User.Login
	}

	themeURLParam := c.Query("theme")
	if themeURLParam == lightName {
		data.User.LightTheme = true
		data.Theme = lightName
	} else if themeURLParam == darkName {
		data.User.LightTheme = false
		data.Theme = darkName
	}

	if hasEditPermissionInFoldersQuery.Result {
		children := []*dtos.NavLink{
			{Text: "Dashboard", Icon: "uil uil-plus-circle", Url: setting.AppSubUrl + "/dashboard/new"},
		}

		if c.OrgRole == m.ROLE_ADMIN || c.OrgRole == m.ROLE_EDITOR {
			children = append(children, &dtos.NavLink{Text: "Folder", SubTitle: "Create a new folder to organize your dashboards", Id: "folder", Icon: "uil uil-folder-plus", Url: setting.AppSubUrl + "/dashboards/folder/new"})
		}

		children = append(children, &dtos.NavLink{Text: "Import", SubTitle: "Import dashboard from file or Grafana.com", Id: "import", Icon: "icon icon-import-dashboard", Url: setting.AppSubUrl + "/dashboard/import"})

		data.NavTree = append(data.NavTree, &dtos.NavLink{
			Text:       "Create",
			Id:         "create",
			Icon:       "uil uil-plus",
			Url:        setting.AppSubUrl + "/dashboard/new",
			Children:   children,
			SortWeight: dtos.WeightCreate,
		})
	}

	dashboardChildNavs := []*dtos.NavLink{
		{Text: "Home", Id: "home", Url: setting.AppSubUrl + "/", Icon: "uil uil-home", HideFromTabs: true},
		{Text: "Divider", Divider: true, Id: "divider", HideFromTabs: true},
		{Text: "Manage", Id: "manage-dashboards", Url: setting.AppSubUrl + "/dashboards", Icon: "uil uil-sitemap"},
		{Text: "Playlists", Id: "playlists", Url: setting.AppSubUrl + "/playlists", Icon: "uil uil-presentation-play"},
		{Text: "Snapshots", Id: "snapshots", Url: setting.AppSubUrl + "/dashboard/snapshots", Icon: "uil uil-camera"},
	}

	data.NavTree = append(data.NavTree, &dtos.NavLink{
		Text:       "Dashboards",
		Id:         "dashboards",
		SubTitle:   "Manage dashboards & folders",
		Icon:       "icon icon-dashboard",
		Url:        setting.AppSubUrl + "/",
		SortWeight: dtos.WeightDashboard,
		Children:   dashboardChildNavs,
	})

	if setting.ExploreEnabled && (c.OrgRole == m.ROLE_ADMIN || c.OrgRole == m.ROLE_EDITOR || setting.ViewersCanEdit) {
		data.NavTree = append(data.NavTree, &dtos.NavLink{
			Text:       "Explore",
			Id:         "explore",
			SubTitle:   "Explore your data",
			Icon:       "uil uil-compass",
			SortWeight: dtos.WeightExplore,
			Url:        setting.AppSubUrl + "/explore",
		})
	}

	if c.IsSignedIn {
		// Only set login if it's different from the name
		var login string
		if c.SignedInUser.Login != c.SignedInUser.NameOrFallback() {
			login = c.SignedInUser.Login
		}
		profileNode := &dtos.NavLink{
			Text:         c.SignedInUser.NameOrFallback(),
			SubTitle:     login,
			Id:           "profile",
			Img:          data.User.GravatarUrl,
			Url:          setting.AppSubUrl + "/profile",
			HideFromMenu: true,
			SortWeight:   dtos.WeightProfile,
			Children: []*dtos.NavLink{
				{Text: "Preferences", Id: "profile-settings", Url: setting.AppSubUrl + "/profile", Icon: "sliders-v"},
				{Text: "Change Password", Id: "change-password", Url: setting.AppSubUrl + "/profile/password", Icon: "clock-nine", HideFromMenu: true},
			},
		}

		if !setting.DisableSignoutMenu {
			// add sign out first
			profileNode.Children = append(profileNode.Children, &dtos.NavLink{
				Text:         "Sign out",
				Id:           "sign-out",
				Url:          setting.AppSubUrl + "/logout",
				Icon:         "uil uil-arrow-from-right",
				Target:       "_self",
				HideFromTabs: true,
			})
		}

		data.NavTree = append(data.NavTree, profileNode)
	}

	if setting.AlertingEnabled && (c.OrgRole == m.ROLE_ADMIN || c.OrgRole == m.ROLE_EDITOR) {
		alertChildNavs := []*dtos.NavLink{
			{Text: "Alert Rules", Id: "alert-list", Url: setting.AppSubUrl + "/alerting/list", Icon: "uil uil-list-ul"},
			{Text: "Notification channels", Id: "channels", Url: setting.AppSubUrl + "/alerting/notifications", Icon: "icon icon-alert-notification"},
		}

		data.NavTree = append(data.NavTree, &dtos.NavLink{
			Text:       "Alerting",
			SubTitle:   "Alert rules & notifications",
			Id:         "alerting",
			Icon:       "uil uil-bell",
			Url:        setting.AppSubUrl + "/alerting/list",
			Children:   alertChildNavs,
			SortWeight: dtos.WeightAlerting,
		})
	}

	enabledPlugins, err := plugins.GetEnabledPlugins(c.OrgId)
	if err != nil {
		return nil, err
	}

	for _, plugin := range enabledPlugins.Apps {
		if plugin.Pinned {
			appLink := &dtos.NavLink{
				Text:       plugin.Name,
				Id:         "plugin-page-" + plugin.Id,
				Url:        plugin.DefaultNavUrl,
				Img:        plugin.Info.Logos.Small,
				SortWeight: dtos.WeightPlugin,
			}

			for _, include := range plugin.Includes {
				if !c.HasUserRole(include.Role) {
					continue
				}

				if include.Type == "page" && include.AddToNav {
					link := &dtos.NavLink{
						Url:  setting.AppSubUrl + "/plugins/" + plugin.Id + "/page/" + include.Slug,
						Text: include.Name,
					}
					appLink.Children = append(appLink.Children, link)
				}

				if include.Type == "dashboard" && include.AddToNav {
					link := &dtos.NavLink{
						Url:  setting.AppSubUrl + "/dashboard/db/" + include.Slug,
						Text: include.Name,
					}
					appLink.Children = append(appLink.Children, link)
				}
			}

			if len(appLink.Children) > 0 && c.OrgRole == m.ROLE_ADMIN {
				appLink.Children = append(appLink.Children, &dtos.NavLink{Divider: true})
				appLink.Children = append(appLink.Children, &dtos.NavLink{Text: "Plugin Config", Icon: "icon icon-setting", Url: setting.AppSubUrl + "/plugins/" + plugin.Id + "/"})
			}

			if len(appLink.Children) > 0 {
				data.NavTree = append(data.NavTree, appLink)
			}
		}
	}

	configNodes := []*dtos.NavLink{}

	if c.OrgRole == m.ROLE_ADMIN {
		configNodes = append(configNodes, &dtos.NavLink{
			Text:        "Data Sources",
			Icon:        "database",
			Description: "Add and configure data sources",
			Id:          "datasources",
			Url:         setting.AppSubUrl + "/datasources",
		})
		configNodes = append(configNodes, &dtos.NavLink{
			Text:        "Users",
			Id:          "users",
			Description: "Manage org members",
			Icon:        "uil uil-user",
			Url:         setting.AppSubUrl + "/org/users",
		})
	}

	if c.OrgRole == m.ROLE_ADMIN || hs.Cfg.EditorsCanAdmin {
		configNodes = append(configNodes, &dtos.NavLink{
			Text:        "Teams",
			Id:          "teams",
			Description: "Manage org groups",
			Icon:        "uil uil-users-alt",
			Url:         setting.AppSubUrl + "/org/teams",
		})
	}

	configNodes = append(configNodes, &dtos.NavLink{
		Text:        "Plugins",
		Id:          "plugins",
		Description: "View and configure plugins",
		Icon:        "uil uil-plug",
		Url:         setting.AppSubUrl + "/plugins",
	})

	if c.OrgRole == m.ROLE_ADMIN {
		configNodes = append(configNodes, &dtos.NavLink{
			Text:        "Preferences",
			Id:          "org-settings",
			Description: "Organization preferences",
			Icon:        "uil uil-sliders-v",
			Url:         setting.AppSubUrl + "/org",
		})
		configNodes = append(configNodes, &dtos.NavLink{
			Text:        "API Keys",
			Id:          "apikeys",
			Description: "Create & manage API keys",
			Icon:        "uil uil-key-skeleton",
			Url:         setting.AppSubUrl + "/org/apikeys",
		})
	}

	data.NavTree = append(data.NavTree, &dtos.NavLink{
		Id:         "cfg",
		Text:       "Configuration",
		SubTitle:   "Organization: " + c.OrgName,
		Icon:       "icon icon-setting",
		Url:        configNodes[0].Url,
		SortWeight: dtos.WeightConfig,
		Children:   configNodes,
	})

	if c.IsGrafanaAdmin {
		adminNavLinks := []*dtos.NavLink{
			{Text: "Users", Id: "global-users", Url: setting.AppSubUrl + "/admin/users", Icon: "uil uil-user"},
			{Text: "Orgs", Id: "global-orgs", Url: setting.AppSubUrl + "/admin/orgs", Icon: "uil uil-building"},
			{Text: "Settings", Id: "server-settings", Url: setting.AppSubUrl + "/admin/settings", Icon: "uil uil-sliders-v"},
			{Text: "Stats", Id: "server-stats", Url: setting.AppSubUrl + "/admin/stats", Icon: "uil uil-graph-bar"},
		}

		if setting.LDAPEnabled {
			adminNavLinks = append(adminNavLinks, &dtos.NavLink{
				Text: "LDAP", Id: "ldap", Url: setting.AppSubUrl + "/admin/ldap", Icon: "uil uil-book",
			})
		}

		data.NavTree = append(data.NavTree, &dtos.NavLink{
			Text:         "Server Admin",
			SubTitle:     "Manage all users & orgs",
			HideFromTabs: true,
			Id:           "admin",
			Icon:         "uil uil-shield",
			Url:          setting.AppSubUrl + "/admin/users",
			SortWeight:   dtos.WeightAdmin,
			Children:     adminNavLinks,
		})
	}

	data.NavTree = append(data.NavTree, &dtos.NavLink{
		Text:         "Help",
		SubTitle:     fmt.Sprintf(`%s v%s (%s)`, setting.ApplicationName, setting.BuildVersion, setting.BuildCommit),
		Id:           "help",
		Url:          "#",
		Icon:         "uil uil-question-circle",
		HideFromMenu: true,
		SortWeight:   dtos.WeightHelp,
		Children: []*dtos.NavLink{
			{Text: "Keyboard shortcuts", Url: "/shortcuts", Icon: "uil uil-keyboard", Target: "_self"},
			{Text: "Community site", Url: "http://community.grafana.com", Icon: "uil uil-comment", Target: "_blank"},
			{Text: "Documentation", Url: "http://docs.grafana.org", Icon: "uil uil-file", Target: "_blank"},
		},
	})

	hs.HooksService.RunIndexDataHooks(&data)

	sort.SliceStable(data.NavTree, func(i, j int) bool {
		return data.NavTree[i].SortWeight < data.NavTree[j].SortWeight
	})
	return &data, nil
}

func (hs *HTTPServer) Index(c *m.ReqContext) {
	data, err := hs.setIndexViewData(c)
	if err != nil {
		c.Handle(500, "Failed to get settings", err)
		return
	}
	c.HTML(200, "index", data)
}

func (hs *HTTPServer) NotFoundHandler(c *m.ReqContext) {
	if c.IsApiRequest() {
		c.JsonApiErr(404, "Not found", nil)
		return
	}

	data, err := hs.setIndexViewData(c)
	if err != nil {
		c.Handle(500, "Failed to get settings", err)
		return
	}

	c.HTML(404, "index", data)
}

func getAppNameBodyClass(validLicense bool) string {
	if validLicense {
		return "app-enterprise"
	}

	return "app-grafana"
}
