# UX Specification

## Model

`Onboarding Center → Recommended Goal → Tour → Guided Action → Verification → Goal Completed`

## Tour Bubble

Each step may contain:

- Arabic title
- Description
- Target element
- Spotlight
- Primary action
- Progress
- Pause
- Skip
- Close

## User Control

### Pause
Preserve the current step and allow resume later.

### Skip
Mark the tour skipped without deleting it.

### Close
Exit the visual overlay while preserving progress.

### Resume
Restore the last persisted step.

## Navigation

Tours can navigate the user to the required route and wait for the target UI to become available.

## Business Verification

A UI click is not completion. The backend/business event must confirm the action.

Example:

`Save → API success → product.created → verify → next step`

## Responsive UX

Support:

- Mobile
- Tablet
- Desktop
- Portrait
- Landscape
- Virtual keyboard
- Scrolling
