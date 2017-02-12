package helpers

import (
	"database/sql"
	"fmt"

	"github.com/pborman/uuid"

	g "github.com/ghmeier/bloodlines/gateways"
	"github.com/ghmeier/coinage/gateways"
	"github.com/ghmeier/coinage/models"
	towncenter "github.com/jakelong95/TownCenter/gateways"
	t "github.com/jakelong95/TownCenter/models"
)

type baseHelper struct {
	sql g.SQL
}

type Roaster struct {
	*baseHelper
	Stripe gateways.Stripe
	TC     towncenter.TownCenterI
}

func NewRoaster(sql g.SQL, stripe gateways.Stripe, towncenter towncenter.TownCenterI) *Roaster {
	return &Roaster{
		baseHelper: &baseHelper{sql: sql},
		Stripe:     stripe,
		TC:         towncenter,
	}
}

func (r *Roaster) Insert(req *models.RoasterRequest) (*models.Roaster, error) {
	user, tRoaster, err := r.roaster(req.UserID)
	if err != nil {
		return nil, err
	}
	stripe, err := r.Stripe.NewAccount(req.Country, user, tRoaster)
	if err != nil {
		return nil, err
	}

	roaster := models.NewRoaster(tRoaster.ID, stripe.ID)
	err = r.sql.Modify(
		"INSERT INTO roaster_account (id, stripeAccountId)VALUE(?, ?, ?)",
		roaster.ID,
		roaster.AccountID,
	)
	if err != nil {
		return nil, err
	}

	roaster.Account = stripe
	return roaster, nil
}

func (r *Roaster) GetByUserID(id uuid.UUID) (*models.Roaster, error) {
	_, roaster, err := r.roaster(id)
	if err != nil {
		return nil, err
	}

	return r.Get(roaster.ID)
}

func (r *Roaster) Get(id uuid.UUID) (*models.Roaster, error) {
	rows, err := r.sql.Select("SELECT id, stripeAccountId FROM roaster_account WHERE roasterId=?", id)
	if err != nil {
		return nil, err
	}

	return r.account(rows)
}

func (r *Roaster) account(rows *sql.Rows) (*models.Roaster, error) {
	roasters, _ := models.RoasterFromSql(rows)
	if len(roasters) < 1 {
		return nil, nil
	}

	roaster := roasters[0]
	stripe, err := r.Stripe.GetAccount(roaster.AccountID)
	if err != nil {
		return nil, err
	}

	roaster.Account = stripe

	return roaster, nil
}

/* roaster returns a towncenter user && roaster by user id. errors otherwise */
func (r *Roaster) roaster(id uuid.UUID) (*t.User, *t.Roaster, error) {
	u, err := r.TC.GetUser(id)
	if err != nil {
		return nil, nil, err
	}

	if u == nil {
		return nil, nil, fmt.Errorf("ERROR: no user for id %s", id.String())
	} else if u.IsRoaster == 0 {
		return nil, nil, fmt.Errorf("ERROR: no roaster for user %s", id.String())
	}

	roaster, err := r.TC.GetRoaster(u.RoasterId)
	if err != nil {
		return nil, nil, err
	}

	if roaster == nil {
		return nil, nil, fmt.Errorf("ERROR: no roaster info for id %s", u.RoasterId)
	}

	return u, roaster, nil
}
