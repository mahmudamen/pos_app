# API Contract

Base: `/api/v1`

## Read State

`GET /api/v1/onboarding`

## Available Tours

`GET /api/v1/onboarding/tours`

## Start

`POST /api/v1/onboarding/tours/{tour_id}/start`

## Progress

`PATCH /api/v1/onboarding/tours/{tour_id}`

## Pause

`POST /api/v1/onboarding/tours/{tour_id}/pause`

## Resume

`POST /api/v1/onboarding/tours/{tour_id}/resume`

## Skip

`POST /api/v1/onboarding/tours/{tour_id}/skip`

## Complete

`POST /api/v1/onboarding/tours/{tour_id}/complete`

## Readiness

`GET /api/v1/onboarding/readiness`

The server must validate user, store, authorization and business state.
