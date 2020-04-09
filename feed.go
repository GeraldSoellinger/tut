package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-mastodon"
	"github.com/rivo/tview"
)

type FeedType uint

const (
	TimelineFeedType FeedType = iota
	ThreadFeedType
	UserFeedType
	NotificationFeedType
	TagFeedType
)

type Feed interface {
	GetFeedList() <-chan string
	LoadNewer() int
	LoadOlder() int
	DrawList()
	DrawToot()
	FeedType() FeedType
	GetSavedIndex() int
	Input(event *tcell.EventKey)
}

func showTootOptions(app *App, status *mastodon.Status, showSensitive bool) (string, string) {
	var line string
	width := app.UI.StatusView.GetTextWidth()
	for i := 0; i < width; i++ {
		line += "-"
	}
	line += "\n"

	shouldDisplay := !status.Sensitive || showSensitive

	var stripped string
	var strippedContent string
	var strippedSpoiler string
	var urls []URL
	var u []URL

	strippedContent, urls = cleanTootHTML(status.Content)

	if status.Sensitive {
		strippedSpoiler, u = cleanTootHTML(status.SpoilerText)
		urls = append(urls, u...)
	}
	if status.Sensitive && !showSensitive {
		strippedSpoiler += "\n" + line
		strippedSpoiler += "Press [s] to show hidden text"
		stripped = strippedSpoiler
	}
	if status.Sensitive && showSensitive {
		stripped = strippedSpoiler + "\n\n" + strippedContent
	}
	if !status.Sensitive {
		stripped = strippedContent
	}

	app.UI.LinkOverlay.SetLinks(urls, status)

	subtleColor := fmt.Sprintf("[#%x]", app.Config.Style.Subtle.Hex())
	special1 := fmt.Sprintf("[#%x]", app.Config.Style.TextSpecial1.Hex())
	special2 := fmt.Sprintf("[#%x]", app.Config.Style.TextSpecial2.Hex())
	var head string
	if status.Reblog != nil {
		if status.Account.DisplayName != "" {
			head += fmt.Sprintf(subtleColor+"%s (%s)\n", status.Account.DisplayName, status.Account.Acct)
		} else {
			head += fmt.Sprintf(subtleColor+"%s\n", status.Account.Acct)
		}
		head += subtleColor + "Boosted\n"
		head += subtleColor + line
		status = status.Reblog
	}

	if status.Account.DisplayName != "" {
		head += fmt.Sprintf(special2+"%s\n", status.Account.DisplayName)
	}
	head += fmt.Sprintf(special1+"%s\n\n", status.Account.Acct)
	output := head
	content := tview.Escape(stripped)
	if content != "" {
		output += content + "\n\n"
	}

	var poll string
	if status.Poll != nil {
		poll += subtleColor + "Poll\n"
		poll += subtleColor + line
		poll += fmt.Sprintf("Number of votes: %d\n\n", status.Poll.VotesCount)
		votes := float64(status.Poll.VotesCount)
		for _, o := range status.Poll.Options {
			res := 0.0
			if votes != 0 {
				res = float64(o.VotesCount) / votes * 100
			}
			poll += fmt.Sprintf("%s - %.2f%% (%d)\n", tview.Escape(o.Title), res, o.VotesCount)
		}
		poll += "\n"
	}

	var media string
	for _, att := range status.MediaAttachments {
		media += subtleColor + line
		media += fmt.Sprintf(subtleColor+"Attached %s\n", att.Type)
		media += fmt.Sprintf("%s\n", att.URL)
	}
	if len(status.MediaAttachments) > 0 {
		media += "\n"
	}

	var card string
	if status.Card != nil {
		card += subtleColor + "Card type: " + status.Card.Type + "\n"
		card += subtleColor + line
		if status.Card.Title != "" {
			card += status.Card.Title + "\n\n"
		}
		desc := strings.TrimSpace(status.Card.Description)
		if desc != "" {
			card += desc + "\n\n"
		}
		card += status.Card.URL
	}

	if shouldDisplay {
		output += poll + media + card
	}

	app.UI.StatusView.ScrollToBeginning()
	var info []string
	if status.Favourited == true {
		info = append(info, ColorKey(app.Config.Style, "Un", "F", "avorite"))
	} else {
		info = append(info, ColorKey(app.Config.Style, "", "F", "avorite"))
	}
	if status.Reblogged == true {
		info = append(info, ColorKey(app.Config.Style, "Un", "B", "boost"))
	} else {
		info = append(info, ColorKey(app.Config.Style, "", "B", "boost"))
	}
	info = append(info, ColorKey(app.Config.Style, "", "T", "hread"))
	info = append(info, ColorKey(app.Config.Style, "", "R", "eply"))
	info = append(info, ColorKey(app.Config.Style, "", "V", "iew"))
	info = append(info, ColorKey(app.Config.Style, "", "U", "ser"))
	if len(status.MediaAttachments) > 0 {
		info = append(info, ColorKey(app.Config.Style, "", "M", "edia"))
	}
	if len(urls)+len(status.Mentions)+len(status.Tags) > 0 {
		info = append(info, ColorKey(app.Config.Style, "", "O", "pen"))
	}

	if status.Account.ID == app.Me.ID {
		info = append(info, ColorKey(app.Config.Style, "", "D", "elete"))
	}

	controls := strings.Join(info, " ")
	return output, controls
}

func drawStatusList(statuses []*mastodon.Status) <-chan string {
	ch := make(chan string)
	go func() {
		today := time.Now()
		ty, tm, td := today.Date()
		for _, s := range statuses {

			sLocal := s.CreatedAt.Local()
			sy, sm, sd := sLocal.Date()
			format := "2006-01-02 15:04"
			if ty == sy && tm == sm && td == sd {
				format = "15:04"
			}
			content := fmt.Sprintf("%s %s", sLocal.Format(format), s.Account.Acct)
			ch <- content
		}
		close(ch)
	}()
	return ch
}

func NewTimelineFeed(app *App, tl TimelineType) *TimelineFeed {
	t := &TimelineFeed{
		app:          app,
		timelineType: tl,
	}
	var err error
	t.statuses, err = t.app.API.GetStatuses(t.timelineType)
	if err != nil {
		t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load timeline toots. Error: %v\n", err))
	}
	return t
}

type TimelineFeed struct {
	app          *App
	timelineType TimelineType
	statuses     []*mastodon.Status
	index        int
	showSpoiler  bool
}

func (t *TimelineFeed) FeedType() FeedType {
	return TimelineFeedType
}

func (t *TimelineFeed) GetCurrentStatus() *mastodon.Status {
	index := t.app.UI.StatusView.GetCurrentItem()
	if index >= len(t.statuses) {
		return nil
	}
	return t.statuses[t.app.UI.StatusView.GetCurrentItem()]
}

func (t *TimelineFeed) GetFeedList() <-chan string {
	return drawStatusList(t.statuses)
}

func (t *TimelineFeed) LoadNewer() int {
	var statuses []*mastodon.Status
	var err error
	if len(t.statuses) == 0 {
		statuses, err = t.app.API.GetStatuses(t.timelineType)
	} else {
		statuses, err = t.app.API.GetStatusesNewer(t.timelineType, t.statuses[0])
	}
	if err != nil {
		t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load new toots. Error: %v\n", err))
		return 0
	}
	if len(statuses) == 0 {
		return 0
	}
	old := t.statuses
	t.statuses = append(statuses, old...)
	return len(statuses)
}

func (t *TimelineFeed) LoadOlder() int {
	var statuses []*mastodon.Status
	var err error
	if len(t.statuses) == 0 {
		statuses, err = t.app.API.GetStatuses(t.timelineType)
	} else {
		statuses, err = t.app.API.GetStatusesOlder(t.timelineType, t.statuses[len(t.statuses)-1])
	}
	if err != nil {
		t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load older toots. Error: %v\n", err))
		return 0
	}
	if len(statuses) == 0 {
		return 0
	}
	t.statuses = append(t.statuses, statuses...)
	return len(statuses)
}

func (t *TimelineFeed) DrawList() {
	t.app.UI.StatusView.SetList(t.GetFeedList())
}

func (t *TimelineFeed) DrawToot() {
	if len(t.statuses) == 0 {
		t.app.UI.StatusView.SetText("")
		t.app.UI.StatusView.SetControls("")
		return
	}
	t.index = t.app.UI.StatusView.GetCurrentItem()
	text, controls := showTootOptions(t.app, t.statuses[t.index], t.showSpoiler)
	t.showSpoiler = false
	t.app.UI.StatusView.SetText(text)
	t.app.UI.StatusView.SetControls(controls)
}

func (t *TimelineFeed) redrawControls() {
	status := t.GetCurrentStatus()
	if status == nil {
		return
	}
	_, controls := showTootOptions(t.app, status, t.showSpoiler)
	t.app.UI.StatusView.SetControls(controls)
}

func (t *TimelineFeed) GetSavedIndex() int {
	return t.index
}

func (t *TimelineFeed) Input(event *tcell.EventKey) {
	status := t.GetCurrentStatus()
	if status == nil {
		return
	}
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 't', 'T':
			t.app.UI.StatusView.AddFeed(
				NewThreadFeed(t.app, status),
			)
		case 'u', 'U':
			t.app.UI.StatusView.AddFeed(
				NewUserFeed(t.app, status.Account),
			)
		case 's', 'S':
			t.showSpoiler = true
			t.DrawToot()
		case 'c', 'C':
			t.app.UI.NewToot()
		case 'o', 'O':
			t.app.UI.ShowLinks()
		case 'r', 'R':
			t.app.UI.Reply(status)
		case 'm', 'M':
			t.app.UI.OpenMedia(status)
		case 'f', 'F':
			index := t.app.UI.StatusView.GetCurrentItem()
			newStatus, err := t.app.API.FavoriteToogle(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't favorite toot. Error: %v\n", err))
				return
			}
			t.statuses[index] = newStatus
			t.redrawControls()

		case 'b', 'B':
			index := t.app.UI.StatusView.GetCurrentItem()
			newStatus, err := t.app.API.BoostToggle(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't boost toot. Error: %v\n", err))
				return
			}
			t.statuses[index] = newStatus
			t.redrawControls()
		case 'd', 'D':
			err := t.app.API.DeleteStatus(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't delete toot. Error: %v\n", err))
			} else {
				status.Card = nil
				status.Sensitive = false
				status.SpoilerText = ""
				status.Favourited = false
				status.MediaAttachments = nil
				status.Reblogged = false
				status.Content = "Deleted"
				t.DrawToot()
			}
		}
	}
}

func NewThreadFeed(app *App, s *mastodon.Status) *ThreadFeed {
	t := &ThreadFeed{
		app: app,
	}
	statuses, index, err := t.app.API.GetThread(s)
	if err != nil {
		t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load thread. Error: %v\n", err))
	}
	t.statuses = statuses
	t.status = s
	t.index = index
	return t
}

type ThreadFeed struct {
	app         *App
	statuses    []*mastodon.Status
	status      *mastodon.Status
	index       int
	showSpoiler bool
}

func (t *ThreadFeed) FeedType() FeedType {
	return ThreadFeedType
}

func (t *ThreadFeed) GetCurrentStatus() *mastodon.Status {
	index := t.app.UI.StatusView.GetCurrentItem()
	if index >= len(t.statuses) {
		return nil
	}
	return t.statuses[t.app.UI.StatusView.GetCurrentItem()]
}

func (t *ThreadFeed) GetFeedList() <-chan string {
	return drawStatusList(t.statuses)
}

func (t *ThreadFeed) LoadNewer() int {
	return 0
}

func (t *ThreadFeed) LoadOlder() int {
	return 0
}

func (t *ThreadFeed) DrawList() {
	t.app.UI.StatusView.SetList(t.GetFeedList())
}

func (t *ThreadFeed) DrawToot() {
	status := t.GetCurrentStatus()
	if status == nil {
		t.app.UI.StatusView.SetText("")
		t.app.UI.StatusView.SetControls("")
		return
	}
	t.index = t.app.UI.StatusView.GetCurrentItem()
	text, controls := showTootOptions(t.app, status, t.showSpoiler)
	t.showSpoiler = false
	t.app.UI.StatusView.SetText(text)
	t.app.UI.StatusView.SetControls(controls)
}

func (t *ThreadFeed) redrawControls() {
	status := t.GetCurrentStatus()
	if status == nil {
		t.app.UI.StatusView.SetText("")
		t.app.UI.StatusView.SetControls("")
		return
	}
	_, controls := showTootOptions(t.app, status, t.showSpoiler)
	t.app.UI.StatusView.SetControls(controls)
}

func (t *ThreadFeed) GetSavedIndex() int {
	return t.index
}

func (t *ThreadFeed) Input(event *tcell.EventKey) {
	status := t.GetCurrentStatus()
	if status == nil {
		return
	}
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 't', 'T':
			if t.status.ID != status.ID {
				t.app.UI.StatusView.AddFeed(
					NewThreadFeed(t.app, status),
				)
			}
		case 'u', 'U':
			t.app.UI.StatusView.AddFeed(
				NewUserFeed(t.app, status.Account),
			)
		case 's', 'S':
			t.showSpoiler = true
			t.DrawToot()
		case 'c', 'C':
			t.app.UI.NewToot()
		case 'o', 'O':
			t.app.UI.ShowLinks()
		case 'r', 'R':
			t.app.UI.Reply(status)
		case 'm', 'M':
			t.app.UI.OpenMedia(status)
		case 'f', 'F':
			index := t.app.UI.StatusView.GetCurrentItem()
			newStatus, err := t.app.API.FavoriteToogle(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't favorite toot. Error: %v\n", err))
				return
			}
			t.statuses[index] = newStatus
			t.redrawControls()

		case 'b', 'B':
			index := t.app.UI.StatusView.GetCurrentItem()
			newStatus, err := t.app.API.BoostToggle(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't boost toot. Error: %v\n", err))
				return
			}
			t.statuses[index] = newStatus
			t.redrawControls()
		case 'd', 'D':
			err := t.app.API.DeleteStatus(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't delete toot. Error: %v\n", err))
			} else {
				status.Card = nil
				status.Sensitive = false
				status.SpoilerText = ""
				status.Favourited = false
				status.MediaAttachments = nil
				status.Reblogged = false
				status.Content = "Deleted"
				t.DrawToot()
			}
		}
	}
}

func NewUserFeed(app *App, a mastodon.Account) *UserFeed {
	u := &UserFeed{
		app: app,
	}
	statuses, err := app.API.GetUserStatuses(a)
	if err != nil {
		u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load user toots. Error: %v\n", err))
		return u
	}
	u.statuses = statuses
	relation, err := app.API.UserRelation(a)
	if err != nil {
		u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load user data. Error: %v\n", err))
		return u
	}
	u.relation = relation
	u.user = a
	return u
}

type UserFeed struct {
	app         *App
	statuses    []*mastodon.Status
	user        mastodon.Account
	relation    *mastodon.Relationship
	index       int
	showSpoiler bool
}

func (u *UserFeed) FeedType() FeedType {
	return UserFeedType
}

func (u *UserFeed) GetCurrentStatus() *mastodon.Status {
	index := u.app.UI.app.UI.StatusView.GetCurrentItem()
	if index > 0 && index-1 >= len(u.statuses) {
		return nil
	}
	return u.statuses[index-1]
}

func (u *UserFeed) GetFeedList() <-chan string {
	ch := make(chan string)
	go func() {
		ch <- "Profile"
		for s := range drawStatusList(u.statuses) {
			ch <- s
		}
		close(ch)
	}()
	return ch
}

func (u *UserFeed) LoadNewer() int {
	var statuses []*mastodon.Status
	var err error
	if len(u.statuses) == 0 {
		statuses, err = u.app.API.GetUserStatuses(u.user)
	} else {
		statuses, err = u.app.API.GetUserStatusesNewer(u.user, u.statuses[0])
	}
	if err != nil {
		u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load new toots. Error: %v\n", err))
		return 0
	}
	if len(statuses) == 0 {
		return 0
	}
	old := u.statuses
	u.statuses = append(statuses, old...)
	return len(statuses)
}

func (u *UserFeed) LoadOlder() int {
	var statuses []*mastodon.Status
	var err error
	if len(u.statuses) == 0 {
		statuses, err = u.app.API.GetUserStatuses(u.user)
	} else {
		statuses, err = u.app.API.GetUserStatusesOlder(u.user, u.statuses[len(u.statuses)-1])
	}
	if err != nil {
		u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load older toots. Error: %v\n", err))
		return 0
	}
	if len(statuses) == 0 {
		return 0
	}
	u.statuses = append(u.statuses, statuses...)
	return len(statuses)
}

func (u *UserFeed) DrawList() {
	u.app.UI.StatusView.SetList(u.GetFeedList())
}

func (u *UserFeed) DrawToot() {
	u.index = u.app.UI.StatusView.GetCurrentItem()

	var text string
	var controls string

	if u.index == 0 {
		n := fmt.Sprintf("[#%x]", u.app.Config.Style.Text.Hex())
		s1 := fmt.Sprintf("[#%x]", u.app.Config.Style.TextSpecial1.Hex())
		s2 := fmt.Sprintf("[#%x]", u.app.Config.Style.TextSpecial2.Hex())

		if u.user.DisplayName != "" {
			text = fmt.Sprintf(s2+"%s\n", u.user.DisplayName)
		}
		text += fmt.Sprintf(s1+"%s\n\n", u.user.Acct)

		text += fmt.Sprintf("Toots %s%d %sFollowers %s%d %sFollowing %s%d\n\n",
			s2, u.user.StatusesCount, n, s2, u.user.FollowersCount, n, s2, u.user.FollowingCount)

		note, urls := cleanTootHTML(u.user.Note)
		text += note + "\n\n"

		for _, f := range u.user.Fields {
			value, fu := cleanTootHTML(f.Value)
			text += fmt.Sprintf("%s%s: %s%s\n", s2, f.Name, n, value)
			urls = append(urls, fu...)
		}

		u.app.UI.LinkOverlay.SetLinks(urls, nil)

		var controlItems []string
		if u.app.Me.ID != u.user.ID {
			if u.relation.Following {
				controlItems = append(controlItems, ColorKey(u.app.Config.Style, "Un", "F", "ollow"))
			} else {
				controlItems = append(controlItems, ColorKey(u.app.Config.Style, "", "F", "ollow"))
			}
			if u.relation.Blocking {
				controlItems = append(controlItems, ColorKey(u.app.Config.Style, "Un", "B", "lock"))
			} else {
				controlItems = append(controlItems, ColorKey(u.app.Config.Style, "", "B", "lock"))
			}
			if u.relation.Muting {
				controlItems = append(controlItems, ColorKey(u.app.Config.Style, "Un", "M", "ute"))
			} else {
				controlItems = append(controlItems, ColorKey(u.app.Config.Style, "", "M", "ute"))
			}
			if len(urls) > 0 {
				controlItems = append(controlItems, ColorKey(u.app.Config.Style, "", "O", "pen"))
			}
			controls = strings.Join(controlItems, " ")
		}

	} else {
		status := u.GetCurrentStatus()
		if status == nil {
			text = ""
			controls = ""
		} else {
			text, controls = showTootOptions(u.app, status, u.showSpoiler)
		}
		u.showSpoiler = false
	}

	u.app.UI.StatusView.SetText(text)
	u.app.UI.StatusView.SetControls(controls)
}

func (u *UserFeed) redrawControls() {
	var controls string
	status := u.GetCurrentStatus()
	if status == nil {
		controls = ""
	} else {
		_, controls = showTootOptions(u.app, status, u.showSpoiler)
	}
	u.app.UI.StatusView.SetControls(controls)
}

func (u *UserFeed) GetSavedIndex() int {
	return u.index
}

func (u *UserFeed) Input(event *tcell.EventKey) {
	index := u.GetSavedIndex()

	if index == 0 {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'f', 'F':
				var relation *mastodon.Relationship
				var err error
				if u.relation.Following {
					relation, err = u.app.API.UnfollowUser(u.user)
				} else {
					relation, err = u.app.API.FollowUser(u.user)
				}
				if err != nil {
					u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't follow/unfollow user. Error: %v\n", err))
					return
				}
				u.relation = relation
				u.DrawToot()
			case 'b', 'B':
				var relation *mastodon.Relationship
				var err error
				if u.relation.Blocking {
					relation, err = u.app.API.UnblockUser(u.user)
				} else {
					relation, err = u.app.API.BlockUser(u.user)
				}
				if err != nil {
					u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't block/unblock user. Error: %v\n", err))
					return
				}
				u.relation = relation
				u.DrawToot()
			case 'm', 'M':
				var relation *mastodon.Relationship
				var err error
				if u.relation.Muting {
					relation, err = u.app.API.UnmuteUser(u.user)
				} else {
					relation, err = u.app.API.MuteUser(u.user)
				}
				if err != nil {
					u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't mute/unmute user. Error: %v\n", err))
					return
				}
				u.relation = relation
				u.DrawToot()
			case 'r', 'R':
				//toots and replies?
			case 'o', 'O':
				u.app.UI.ShowLinks()
			}
		}
		return
	}

	if event.Key() == tcell.KeyRune {
		status := u.GetCurrentStatus()
		if status == nil {
			return
		}
		switch event.Rune() {
		case 't', 'T':
			u.app.UI.StatusView.AddFeed(
				NewThreadFeed(u.app, status),
			)
		case 'u', 'U':
			if u.user.ID != status.Account.ID {
				u.app.UI.StatusView.AddFeed(
					NewUserFeed(u.app, status.Account),
				)
			}
		case 's', 'S':
			u.showSpoiler = true
			u.DrawToot()
		case 'c', 'C':
			u.app.UI.NewToot()
		case 'o', 'O':
			u.app.UI.ShowLinks()
		case 'r', 'R':
			u.app.UI.Reply(status)
		case 'm', 'M':
			u.app.UI.OpenMedia(status)
		case 'f', 'F':
			index := u.app.UI.StatusView.GetCurrentItem()
			newStatus, err := u.app.API.FavoriteToogle(status)
			if err != nil {
				u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't favorite toot. Error: %v\n", err))
				return
			}
			u.statuses[index-1] = newStatus
			u.redrawControls()

		case 'b', 'B':
			index := u.app.UI.StatusView.GetCurrentItem()
			newStatus, err := u.app.API.BoostToggle(status)
			if err != nil {
				u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't boost toot. Error: %v\n", err))
				return
			}
			u.statuses[index-1] = newStatus
			u.redrawControls()
		case 'd', 'D':
			err := u.app.API.DeleteStatus(status)
			if err != nil {
				u.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't delete toot. Error: %v\n", err))
			} else {
				status.Card = nil
				status.Sensitive = false
				status.SpoilerText = ""
				status.Favourited = false
				status.MediaAttachments = nil
				status.Reblogged = false
				status.Content = "Deleted"
				u.DrawToot()
			}
		}
	}
}

func NewNoticifationsFeed(app *App) *NotificationsFeed {
	n := &NotificationsFeed{
		app: app,
	}
	n.notifications, _ = n.app.API.GetNotifications()
	return n
}

type NotificationsFeed struct {
	app           *App
	timelineType  TimelineType
	notifications []*mastodon.Notification
	index         int
	showSpoiler   bool
}

func (n *NotificationsFeed) FeedType() FeedType {
	return NotificationFeedType
}

func (n *NotificationsFeed) GetCurrentNotification() *mastodon.Notification {
	index := n.app.UI.StatusView.GetCurrentItem()
	if index >= len(n.notifications) {
		return nil
	}
	return n.notifications[index]
}

func (n *NotificationsFeed) GetFeedList() <-chan string {
	ch := make(chan string)
	notifications := n.notifications
	go func() {
		today := time.Now()
		ty, tm, td := today.Date()
		for _, item := range notifications {
			sLocal := item.CreatedAt.Local()
			sy, sm, sd := sLocal.Date()
			format := "2006-01-02 15:04"
			if ty == sy && tm == sm && td == sd {
				format = "15:04"
			}
			content := fmt.Sprintf("%s %s", sLocal.Format(format), item.Account.Acct)
			ch <- content
		}
		close(ch)
	}()
	return ch
}

func (n *NotificationsFeed) LoadNewer() int {
	var notifications []*mastodon.Notification
	var err error
	if len(n.notifications) == 0 {
		notifications, err = n.app.API.GetNotifications()
	} else {
		notifications, err = n.app.API.GetNotificationsNewer(n.notifications[0])
	}
	if err != nil {
		n.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load new toots. Error: %v\n", err))
		return 0
	}
	if len(notifications) == 0 {
		return 0
	}
	old := n.notifications
	n.notifications = append(notifications, old...)
	return len(notifications)
}

func (n *NotificationsFeed) LoadOlder() int {
	var notifications []*mastodon.Notification
	var err error
	if len(n.notifications) == 0 {
		notifications, err = n.app.API.GetNotifications()
	} else {
		notifications, err = n.app.API.GetNotificationsOlder(n.notifications[len(n.notifications)-1])
	}
	if err != nil {
		n.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load older toots. Error: %v\n", err))
		return 0
	}
	if len(notifications) == 0 {
		return 0
	}
	n.notifications = append(n.notifications, notifications...)
	return len(notifications)
}

func (n *NotificationsFeed) DrawList() {
	n.app.UI.StatusView.SetList(n.GetFeedList())
}

func (n *NotificationsFeed) DrawToot() {
	n.index = n.app.UI.StatusView.GetCurrentItem()
	notification := n.GetCurrentNotification()
	if notification == nil {
		n.app.UI.StatusView.SetText("")
		n.app.UI.StatusView.SetControls("")
		return
	}
	var text string
	var controls string
	defer func() { n.showSpoiler = false }()

	switch notification.Type {
	case "follow":
		text = SublteText(n.app.Config.Style, FormatUsername(notification.Account)+" started following you\n\n")
		controls = ColorKey(n.app.Config.Style, "", "U", "ser")
	case "favourite":
		pre := SublteText(n.app.Config.Style, FormatUsername(notification.Account)+" favorited your toot") + "\n\n"
		text, controls = showTootOptions(n.app, notification.Status, n.showSpoiler)
		text = pre + text
	case "reblog":
		pre := SublteText(n.app.Config.Style, FormatUsername(notification.Account)+" boosted your toot") + "\n\n"
		text, controls = showTootOptions(n.app, notification.Status, n.showSpoiler)
		text = pre + text
	case "mention":
		pre := SublteText(n.app.Config.Style, FormatUsername(notification.Account)+" mentioned you") + "\n\n"
		text, controls = showTootOptions(n.app, notification.Status, n.showSpoiler)
		text = pre + text
	case "poll":
		pre := SublteText(n.app.Config.Style, "A poll of yours or one you participated in has ended") + "\n\n"
		text, controls = showTootOptions(n.app, notification.Status, n.showSpoiler)
		text = pre + text
	}

	n.app.UI.StatusView.SetText(text)
	n.app.UI.StatusView.SetControls(controls)
}

func (n *NotificationsFeed) redrawControls() {
	notification := n.GetCurrentNotification()
	if notification == nil {
		n.app.UI.StatusView.SetControls("")
		return
	}
	switch notification.Type {
	case "favourite", "reblog", "mention", "poll":
		_, controls := showTootOptions(n.app, notification.Status, n.showSpoiler)
		n.app.UI.StatusView.SetControls(controls)
	}
}

func (n *NotificationsFeed) GetSavedIndex() int {
	return n.index
}

func (n *NotificationsFeed) Input(event *tcell.EventKey) {
	notification := n.GetCurrentNotification()
	if notification == nil {
		return
	}
	if notification.Type == "follow" {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'u', 'U':
				n.app.UI.StatusView.AddFeed(
					NewUserFeed(n.app, notification.Account),
				)
			}
		}
		return
	}

	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 't', 'T':
			n.app.UI.StatusView.AddFeed(
				NewThreadFeed(n.app, notification.Status),
			)
		case 'u', 'U':
			n.app.UI.StatusView.AddFeed(
				NewUserFeed(n.app, notification.Account),
			)
		case 's', 'S':
			n.showSpoiler = true
			n.DrawToot()
		case 'c', 'C':
			n.app.UI.NewToot()
		case 'o', 'O':
			n.app.UI.ShowLinks()
		case 'r', 'R':
			n.app.UI.Reply(notification.Status)
		case 'm', 'M':
			n.app.UI.OpenMedia(notification.Status)
		case 'f', 'F':
			index := n.app.UI.StatusView.GetCurrentItem()
			status, err := n.app.API.FavoriteToogle(notification.Status)
			if err != nil {
				n.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't favorite toot. Error: %v\n", err))
				return
			}
			n.notifications[index].Status = status
			n.redrawControls()

		case 'b', 'B':
			index := n.app.UI.StatusView.GetCurrentItem()
			status, err := n.app.API.BoostToggle(notification.Status)
			if err != nil {
				n.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't boost toot. Error: %v\n", err))
				return
			}
			n.notifications[index].Status = status
			n.redrawControls()
		case 'd', 'D':
			err := n.app.API.DeleteStatus(notification.Status)
			if err != nil {
				n.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't delete toot. Error: %v\n", err))
			} else {
				notification.Status.Card = nil
				notification.Status.Sensitive = false
				notification.Status.SpoilerText = ""
				notification.Status.Favourited = false
				notification.Status.MediaAttachments = nil
				notification.Status.Reblogged = false
				notification.Status.Content = "Deleted"
				n.DrawToot()
			}
		}
	}
}

func NewTagFeed(app *App, tag string) *TagFeed {
	t := &TagFeed{
		app: app,
		tag: tag,
	}
	t.statuses, _ = t.app.API.GetTags(tag)
	return t
}

type TagFeed struct {
	app         *App
	tag         string
	statuses    []*mastodon.Status
	index       int
	showSpoiler bool
}

func (t *TagFeed) FeedType() FeedType {
	return TagFeedType
}

func (t *TagFeed) GetCurrentStatus() *mastodon.Status {
	index := t.app.UI.StatusView.GetCurrentItem()
	if index >= len(t.statuses) {
		return nil
	}
	return t.statuses[t.app.UI.StatusView.GetCurrentItem()]
}

func (t *TagFeed) GetFeedList() <-chan string {
	return drawStatusList(t.statuses)
}

func (t *TagFeed) LoadNewer() int {
	var statuses []*mastodon.Status
	var err error
	if len(t.statuses) == 0 {
		statuses, err = t.app.API.GetTags(t.tag)
	} else {
		statuses, err = t.app.API.GetTagsNewer(t.tag, t.statuses[0])
	}
	if err != nil {
		t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load new toots. Error: %v\n", err))
		return 0
	}
	if len(statuses) == 0 {
		return 0
	}
	old := t.statuses
	t.statuses = append(statuses, old...)
	return len(statuses)
}

func (t *TagFeed) LoadOlder() int {
	var statuses []*mastodon.Status
	var err error
	if len(t.statuses) == 0 {
		statuses, err = t.app.API.GetTags(t.tag)
	} else {
		statuses, err = t.app.API.GetTagsOlder(t.tag, t.statuses[len(t.statuses)-1])
	}
	if err != nil {
		t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't load older toots. Error: %v\n", err))
		return 0
	}
	if len(statuses) == 0 {
		return 0
	}
	t.statuses = append(t.statuses, statuses...)
	return len(statuses)
}

func (t *TagFeed) DrawList() {
	t.app.UI.StatusView.SetList(t.GetFeedList())
}

func (t *TagFeed) DrawToot() {
	if len(t.statuses) == 0 {
		t.app.UI.StatusView.SetText("")
		t.app.UI.StatusView.SetControls("")
		return
	}
	t.index = t.app.UI.StatusView.GetCurrentItem()
	text, controls := showTootOptions(t.app, t.statuses[t.index], t.showSpoiler)
	t.showSpoiler = false
	t.app.UI.StatusView.SetText(text)
	t.app.UI.StatusView.SetControls(controls)
}

func (t *TagFeed) redrawControls() {
	status := t.GetCurrentStatus()
	if status == nil {
		return
	}
	_, controls := showTootOptions(t.app, status, t.showSpoiler)
	t.app.UI.StatusView.SetControls(controls)
}

func (t *TagFeed) GetSavedIndex() int {
	return t.index
}

func (t *TagFeed) Input(event *tcell.EventKey) {
	status := t.GetCurrentStatus()
	if status == nil {
		return
	}
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 't', 'T':
			t.app.UI.StatusView.AddFeed(
				NewThreadFeed(t.app, status),
			)
		case 'u', 'U':
			t.app.UI.StatusView.AddFeed(
				NewUserFeed(t.app, status.Account),
			)
		case 's', 'S':
			t.showSpoiler = true
			t.DrawToot()
		case 'c', 'C':
			t.app.UI.NewToot()
		case 'o', 'O':
			t.app.UI.ShowLinks()
		case 'r', 'R':
			t.app.UI.Reply(status)
		case 'm', 'M':
			t.app.UI.OpenMedia(status)
		case 'f', 'F':
			index := t.app.UI.StatusView.GetCurrentItem()
			newStatus, err := t.app.API.FavoriteToogle(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't favorite toot. Error: %v\n", err))
				return
			}
			t.statuses[index] = newStatus
			t.redrawControls()

		case 'b', 'B':
			index := t.app.UI.StatusView.GetCurrentItem()
			newStatus, err := t.app.API.BoostToggle(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't boost toot. Error: %v\n", err))
				return
			}
			t.statuses[index] = newStatus
			t.redrawControls()
		case 'd', 'D':
			err := t.app.API.DeleteStatus(status)
			if err != nil {
				t.app.UI.CmdBar.ShowError(fmt.Sprintf("Couldn't delete toot. Error: %v\n", err))
			} else {
				status.Card = nil
				status.Sensitive = false
				status.SpoilerText = ""
				status.Favourited = false
				status.MediaAttachments = nil
				status.Reblogged = false
				status.Content = "Deleted"
				t.DrawToot()
			}
		}
	}
}
