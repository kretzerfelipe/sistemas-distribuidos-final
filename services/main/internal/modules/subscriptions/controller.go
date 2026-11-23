package subscriptions

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"futhub/main/internal/api"
	"futhub/main/internal/entities"
	"futhub/main/internal/platform/httpx"
	"futhub/main/internal/platform/routing"
)

type controller struct {
	uc UseCase
}

func newController(uc UseCase) *controller {
	return &controller{uc: uc}
}

func (c *controller) routes() []routing.Route {
	subscribers := routing.Roles(entities.RoleUser, entities.RoleTeam)

	return []routing.Route{
		{Method: http.MethodPost, Path: "/subscriptions", Handler: c.subscribe, Auth: subscribers},
		{Method: http.MethodGet, Path: "/subscriptions/mine", Handler: c.getMine, Auth: subscribers},
		{Method: http.MethodGet, Path: "/subscriptions/{id}", Handler: c.getByID, Auth: routing.Authenticated()},
	}
}

func (c *controller) subscribe(w http.ResponseWriter, r *http.Request) {
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	subscription, err := c.uc.Subscribe(r.Context(), caller)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/subscriptions/%s", subscription.ID))
	httpx.WriteJSON(w, http.StatusAccepted, NewSubscriptionResponse(subscription))
}

func (c *controller) getMine(w http.ResponseWriter, r *http.Request) {
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	subscription, err := c.uc.GetMine(r.Context(), caller)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewSubscriptionResponse(subscription))
}

func (c *controller) getByID(w http.ResponseWriter, r *http.Request) {
	subscriptionID, err := httpx.GetVariable[uuid.UUID](r, "id")
	if err != nil {
		api.WriteError(w, err)
		return
	}
	caller, err := api.GetCaller(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	subscription, err := c.uc.GetByID(r.Context(), caller, subscriptionID)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, NewSubscriptionResponse(subscription))
}
