package opsconsole

import (
	"context"
	"fmt"

	"m31labs.dev/fluffyui/fluffy"
	"m31labs.dev/fluffyui/runtime"
)

// appOptions are the AppOptions the console runs with.
//
// HighContrast is on because this is an ops tool: an operator reads status
// colours to decide whether a merchant is locked out, and that judgement has to
// survive a low-quality SSH terminal or a colourblind reader. Focus
// registration is automatic so the framework scans the tree and keeps tab order
// correct as panes are swapped in and out.
func appOptions(root runtime.Widget) []fluffy.AppOption {
	return []fluffy.AppOption{
		fluffy.WithRoot(root),
		fluffy.WithStylesheet(ConsoleStylesheet()),
		fluffy.WithHighContrast(),
		fluffy.WithFocusIndicator("▌"),
		fluffy.WithFocusRegistration(runtime.FocusRegistrationAuto),
	}
}

// Run builds the app around an already-assembled root widget, hands the console
// the app's UI scheduler, optionally opens the agent control socket, and blocks
// until the operator quits.
//
// mcpAddr is passed rather than configured inside Run so the decision to expose
// a privileged view of every merchant stays with the caller, in main, where it
// is announced on the terminal.
func Run(ctx context.Context, console *Console, root runtime.Widget, mcpAddr string) error {
	app, err := fluffy.NewApp(appOptions(root)...)
	if err != nil {
		return fmt.Errorf("opsconsole: start app: %w", err)
	}
	// Now that a scheduler exists, move the fetch-completion repaints onto the
	// render thread. Done here rather than in New because the console is built
	// before the app it will live in.
	console.bindScheduler(app.StateScheduler())
	defer console.dispose()

	if mcpAddr != "" {
		closeMCP, err := EnableMCP(app, mcpAddr)
		if err != nil {
			return err
		}
		if closeMCP != nil {
			defer closeMCP()
		}
	}
	return app.Run(ctx)
}

// EnableMCP exposes the console to an AI agent over a unix socket, so an agent
// can read the current screen and drive the same controls an operator uses.
//
// It is opt-in through the MCP address, and it is never enabled implicitly: the
// console is a privileged view of every merchant on the platform, so opening a
// control socket requires an operator to ask for it by name. A caller that
// supplies an address gets raw screen text enabled, because an agent that
// cannot read the screen can only guess.
func EnableMCP(app *runtime.App, addr string) (func() error, error) {
	if addr == "" {
		return nil, nil
	}
	closer, err := app.EnableMCP(runtime.MCPOptions{
		Transport: "unix",
		Addr:      addr,
		AllowText: true,
	})
	if err != nil {
		return nil, fmt.Errorf("opsconsole: enable MCP on %s: %w", addr, err)
	}
	return closer.Close, nil
}
