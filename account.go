package gondole

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sendgrid/rest"
)

// getAccountsOptions contains option fields for POST and DELETE API calls
type getAccountsOptions struct {
	// The ID is used for most commands
	ID int

	// The following fields are used when searching for accounts
	Q     string
	Limit int
}

// getSingleAccount returns an account entity
// The target can be "account", "verify_credentials", "follow", "unfollow",
// "block", "unblock", "mute", "unmute", "follow_requests/authorize" or
// "follow_requests/reject".
// The id is optional and depends on the target.
func (g *Client) getSingleAccount(target string, id int) (*Account, error) {
	var endPoint string
	method := rest.Get
	strID := strconv.Itoa(id)

	switch target {
	case "account":
		endPoint = "accounts/" + strID
	case "verify_credentials":
		endPoint = "accounts/verify_credentials"
	case "follow", "unfollow", "block", "unblock", "mute", "unmute":
		endPoint = "accounts/" + strID + "/" + target
		method = rest.Post
	case "follow_requests/authorize", "follow_requests/reject":
		// The documentation is incorrect, the endpoint actually
		// is "follow_requests/:id/{authorize|reject}"
		endPoint = target[:16] + strID + "/" + target[16:]
		method = rest.Post
	default:
		return nil, ErrInvalidParameter
	}

	req := g.prepareRequest(endPoint)
	req.Method = method

	r, err := rest.API(req)
	if err != nil {
		return nil, fmt.Errorf("getAccount (%s): %s", target, err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var account Account
	err = json.Unmarshal([]byte(r.Body), &account)
	if err != nil {
		return nil, fmt.Errorf("getAccount (%s) API: %s", target, err.Error())
	}
	return &account, nil
}

// getMultipleAccounts returns a list of account entities
// The target can be "followers", "following", "search", "blocks", "mutes",
// "follow_requests".
// The id is optional and depends on the target.
func (g *Client) getMultipleAccounts(target string, opts *getAccountsOptions) ([]Account, error) {
	var endPoint string
	switch target {
	case "followers", "following":
		if opts == nil || opts.ID < 1 {
			return []Account{}, ErrInvalidID
		}
		endPoint = "accounts/" + strconv.Itoa(opts.ID) + "/" + target
	case "follow_requests", "blocks", "mutes":
		endPoint = target
	case "search":
		if opts == nil || opts.Q == "" {
			return []Account{}, ErrInvalidParameter
		}
		endPoint = "accounts/" + target
	default:
		return nil, ErrInvalidParameter
	}

	req := g.prepareRequest(endPoint)

	// Handle target-specific query parameters
	if target == "search" {
		req.QueryParams["q"] = opts.Q
		if opts.Limit > 0 {
			req.QueryParams["limit"] = strconv.Itoa(opts.Limit)
		}
	}

	r, err := rest.API(req)
	if err != nil {
		return nil, fmt.Errorf("getAccount (%s): %s", target, err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var accounts []Account
	err = json.Unmarshal([]byte(r.Body), &accounts)
	if err != nil {
		return nil, fmt.Errorf("getAccount (%s) API: %s", target, err.Error())
	}
	return accounts, nil
}

// GetAccount returns an account entity
// The returned value can be nil if there is an error or if the
// requested ID does not exist.
func (g *Client) GetAccount(id int) (*Account, error) {
	account, err := g.getSingleAccount("account", id)
	if err != nil {
		return nil, err
	}
	if account != nil && account.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return account, nil
}

// GetCurrentAccount returns the current user account
func (g *Client) GetCurrentAccount() (*Account, error) {
	account, err := g.getSingleAccount("verify_credentials", 0)
	if err != nil {
		return nil, err
	}
	if account != nil && account.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return account, nil
}

// GetAccountFollowers returns the list of accounts following a given account
func (g *Client) GetAccountFollowers(accountID int) ([]Account, error) {
	o := &getAccountsOptions{ID: accountID}
	return g.getMultipleAccounts("followers", o)
}

// GetAccountFollowing returns the list of accounts a given account is following
func (g *Client) GetAccountFollowing(accountID int) ([]Account, error) {
	o := &getAccountsOptions{ID: accountID}
	return g.getMultipleAccounts("following", o)
}

// FollowAccount follows an account
func (g *Client) FollowAccount(id int) error {
	account, err := g.getSingleAccount("follow", id)
	if err != nil {
		return err
	}
	if account != nil && account.ID != id {
		return ErrEntityNotFound
	}
	return nil
}

// UnfollowAccount unfollows an account
func (g *Client) UnfollowAccount(id int) error {
	account, err := g.getSingleAccount("unfollow", id)
	if err != nil {
		return err
	}
	if account != nil && account.ID != id {
		return ErrEntityNotFound
	}
	return nil
}

// FollowRemoteAccount follows a remote account
// The parameter 'id' is a URI (username@domain).
func (g *Client) FollowRemoteAccount(id string) (*Account, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	req := g.prepareRequest("follows")
	req.Method = rest.Post
	req.QueryParams["uri"] = id
	r, err := rest.API(req)
	if err != nil {
		return nil, fmt.Errorf("FollowRemoteAccount: %s", err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var account Account
	err = json.Unmarshal([]byte(r.Body), &account)
	if err != nil {
		return nil, fmt.Errorf("FollowRemoteAccount API: %s", err.Error())
	}
	if account.ID == 0 {
		return nil, ErrEntityNotFound
	}
	return &account, nil
}

// BlockAccount blocks an account
func (g *Client) BlockAccount(id int) error {
	account, err := g.getSingleAccount("block", id)
	if err != nil {
		return err
	}
	if account != nil && account.ID != id {
		return ErrEntityNotFound
	}
	return nil
}

// UnblockAccount unblocks an account
func (g *Client) UnblockAccount(id int) error {
	account, err := g.getSingleAccount("unblock", id)
	if err != nil {
		return err
	}
	if account != nil && account.ID != id {
		return ErrEntityNotFound
	}
	return nil
}

// MuteAccount mutes an account
func (g *Client) MuteAccount(id int) error {
	account, err := g.getSingleAccount("mute", id)
	if err != nil {
		return err
	}
	if account != nil && account.ID != id {
		return ErrEntityNotFound
	}
	return nil
}

// UnmuteAccount unmutes an account
func (g *Client) UnmuteAccount(id int) error {
	account, err := g.getSingleAccount("unmute", id)
	if err != nil {
		return err
	}
	if account != nil && account.ID != id {
		return ErrEntityNotFound
	}
	return nil
}

// SearchAccounts returns a list of accounts matching the query string
// The limit parameter is optional (can be 0).
func (g *Client) SearchAccounts(query string, limit int) ([]Account, error) {
	o := &getAccountsOptions{Q: query, Limit: limit}
	return g.getMultipleAccounts("search", o)
}

// GetBlockedAccounts returns the list of blocked accounts
func (g *Client) GetBlockedAccounts() ([]Account, error) {
	return g.getMultipleAccounts("blocks", nil)
}

// GetMutedAccounts returns the list of muted accounts
func (g *Client) GetMutedAccounts() ([]Account, error) {
	return g.getMultipleAccounts("mutes", nil)
}

// GetAccountFollowRequests returns the list of follow requests accounts
func (g *Client) GetAccountFollowRequests() ([]Account, error) {
	return g.getMultipleAccounts("follow_requests", nil)
}

// GetAccountRelationships returns a list of relationship entities for the given accounts
// NOTE: Currently it doesn't seem to work with several items.
func (g *Client) GetAccountRelationships(accountIDs []int) ([]Relationship, error) {
	if len(accountIDs) < 1 {
		return nil, ErrInvalidID
	}

	req := g.prepareRequest("accounts/relationships")

	if len(accountIDs) > 1 { // XXX
		return nil, fmt.Errorf("accounts/relationships currently does not work with more than 1 ID")
	}
	req.QueryParams["id"] = strconv.Itoa(accountIDs[0])
	/*
		for i, id := range accountIDList {
			qID := fmt.Sprintf("id[%d]", i+1)
			req.QueryParams[qID] = strconv.Itoa(id)
		}
	*/

	r, err := rest.API(req)
	if err != nil {
		return nil, fmt.Errorf("GetAccountRelationships: %s", err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var rl []Relationship
	err = json.Unmarshal([]byte(r.Body), &rl)
	if err != nil {
		return nil, fmt.Errorf("accounts/relationships API: %s", err.Error())
	}
	return rl, nil
}

// GetAccountStatuses returns a list of status entities for the given account
// If onlyMedia is true, returns only statuses that have media attachments.
// If excludeReplies is true, skip statuses that reply to other statuses.
func (g *Client) GetAccountStatuses(accountID int, onlyMedia, excludeReplies bool) ([]Status, error) {
	if accountID < 1 {
		return nil, ErrInvalidID
	}

	endPoint := "accounts/" + strconv.Itoa(accountID) + "/" + "statuses"
	req := g.prepareRequest(endPoint)

	if onlyMedia {
		req.QueryParams["only_media"] = "true"
	}
	if excludeReplies {
		req.QueryParams["exclude_replies"] = "true"
	}

	r, err := rest.API(req)
	if err != nil {
		return nil, fmt.Errorf("GetAccountStatuses: %s", err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var sl []Status
	err = json.Unmarshal([]byte(r.Body), &sl)
	if err != nil {
		return nil, fmt.Errorf("accounts/statuses API: %s", err.Error())
	}
	return sl, nil
}

// FollowRequestAuthorize authorizes or rejects an account follow-request
func (g *Client) FollowRequestAuthorize(accountID int, authorize bool) error {
	endPoint := "follow_requests/reject"
	if authorize {
		endPoint = "follow_requests/authorize"
	}
	_, err := g.getSingleAccount(endPoint, accountID)
	return err
}
