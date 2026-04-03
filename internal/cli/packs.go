package cli

import (
	"context"
	"fmt"
	"io"
)

// RunPacksList lists all quiz packs and indicates which are active.
func (a *App) RunPacksList(w io.Writer) error {
	ctx := context.Background()

	packs, err := a.repo.ListQuizPacks(ctx)
	if err != nil {
		return fmt.Errorf("listing packs: %w", err)
	}

	if len(packs) == 0 {
		fmt.Fprintln(w, "No quiz packs found.")
		return nil
	}

	prefs, err := a.getPreferences(ctx)
	if err != nil {
		return err
	}

	activeSet := make(map[string]bool, len(prefs.ActivePackIDs))
	for _, id := range prefs.ActivePackIDs {
		activeSet[id] = true
	}

	for _, p := range packs {
		status := "  "
		if activeSet[p.ID] {
			status = "* "
		}
		fmt.Fprintf(w, "%s%s - %s (%d questions)\n", status, p.ID, p.Name, len(p.Questions))
	}

	return nil
}

// RunPacksActivate adds a pack ID to the user's active pack IDs.
func (a *App) RunPacksActivate(packID string, w io.Writer) error {
	ctx := context.Background()

	// Verify the pack exists.
	if _, err := a.repo.GetQuizPack(ctx, packID); err != nil {
		return fmt.Errorf("pack not found: %s", packID)
	}

	prefs, err := a.getPreferences(ctx)
	if err != nil {
		return err
	}

	// Check if already active.
	for _, id := range prefs.ActivePackIDs {
		if id == packID {
			fmt.Fprintf(w, "Pack %s is already active.\n", packID)
			return nil
		}
	}

	prefs.ActivePackIDs = append(prefs.ActivePackIDs, packID)
	if err := a.savePreferences(ctx, *prefs); err != nil {
		return err
	}

	fmt.Fprintf(w, "Activated pack: %s\n", packID)
	return nil
}

// RunPacksDeactivate removes a pack ID from the user's active pack IDs.
func (a *App) RunPacksDeactivate(packID string, w io.Writer) error {
	ctx := context.Background()

	prefs, err := a.getPreferences(ctx)
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(prefs.ActivePackIDs))
	found := false
	for _, id := range prefs.ActivePackIDs {
		if id == packID {
			found = true
			continue
		}
		filtered = append(filtered, id)
	}

	if !found {
		fmt.Fprintf(w, "Pack %s is not active.\n", packID)
		return nil
	}

	prefs.ActivePackIDs = filtered
	if err := a.savePreferences(ctx, *prefs); err != nil {
		return err
	}

	fmt.Fprintf(w, "Deactivated pack: %s\n", packID)
	return nil
}
