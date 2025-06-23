# Tree View Implementation Plan

This document outlines the tasks required to implement the tree view for ads on the search/home page.

## Backend

1.  **Tree Hierarchy Definition:**
    - The hierarchy will be: Make -> Year -> Model -> Engine -> Category -> SubCategory.
    - Ads can be attached at any level.

2.  **API Endpoint (`/tree/...`):**
    - Create a new handler in `handlers/search.go` for the path prefix `/tree/`.
    - It will be mounted on a router. `r.Get("/tree/*", h.TreeView)`
    - The handler will parse the path components to determine the context (e.g. `Make`, `Year`, etc.).
    - It will also parse query parameters from the request, like the search query `q`.

3.  **Database Logic:**
    - Create new functions in `part/part.go` to query for tree nodes.
    - `GetMakes(query string) ([]string, error)`: Get all makes for ads matching the query.
    - `GetYears(query string, make string) ([]string, error)`: Get all years for a make. An ad with a year range should be included for each year in its range.
    - And so on for `GetModels`, `GetEngines`, `GetCategories`, `GetSubCategories`.
    - `GetAdsForNode(...)`: Get ads that should be displayed at a specific node in the tree.

4.  **UI Component Rendering (Backend):**
    - Create a `ui.TreeNode` gomponent in a new `ui/tree.go` file.
    - This component will render a single node in the tree.
    - It will include the `+` button with the correct `hx-get` attribute to load children.
    - The handler for `/tree/...` will call these DB functions and render a list of `ui.TreeNode` components.

## Frontend

1.  **View Toggle on Search Page:**
    - In `ui/search.go`, modify `SearchPage` component.
    - Add "List View" and "Tree View" buttons.
    - Add two divs, one for list (`#list-view`) and one for tree (`#tree-view`).
    - The buttons will toggle the visibility of these divs. A little bit of JS might be needed, or clever use of htmx to add/remove a `hidden` class.

2.  **Initial Tree View:**
    - The `#tree-view` div will be initially empty.
    - It will be populated with the top-level makes via htmx.
    - `<div id="tree-view" hx-get="/tree" hx-trigger="load" hx-swap="innerHTML"></div>`
    - The search input should trigger a reload of the tree view:
      `<input ... hx-get="/tree" hx-target="#tree-view" hx-trigger="keyup changed delay:500ms">`

3.  **Node Expansion/Collapse:**
    - The `+` button on a `TreeNode` will fetch children.
    - `hx-get` will point to the next level of the tree, e.g., `/tree/Ford`.
    - `hx-target` will be the `div` for children within the current `TreeNode`.
    - `hx-swap` will be `innerHTML`.
    - After content is loaded, the `+` will be replaced by a `-`. We can do this by having the server return the `-` button as part of the response, which replaces the `+` button.
    - To make collapse a client-side operation to preserve subtree state:
        - The `-` button won't have `hx-*` attributes. It will trigger a small javascript function to toggle a `hidden` class on the children `div`, and to change itself back to a `+`.
        - The `+` button, when clicked again, will just un-hide the content, not re-fetch it.
        - This means we need to distinguish between a node that has never been expanded, and one that has been expanded and then collapsed.
        - Alternative: the `-` button swaps the children content with an empty string, and replaces itself with the original `+` button. This is simpler and doesn't require JS, but loses the subtree state. Given the user's wish, the JS approach is better. We can add a little script to the page.

4.  **Ad Display:**
    - When the backend handler determines that a node should display ads, it will query for them and render them using the existing `ui.AdCard` component. This will be part of the response to an expansion request.

## Data Structures

- Review `ad.Ad`, `vehicle.Vehicle`, and `part.Part` to ensure they support this hierarchical view.
- The `ads` table in `schema.sql` has `year_start` and `year_end`, which is good.

## File Changes

- `TODO-tree.md`: Create this file.
- `handlers/search.go`: Add `TreeView` handler.
- `main.go`: Add route for `/tree/*`.
- `ui/search.go`: Update `SearchPage` with view toggles and tree container.
- `ui/tree.go`: New file for `TreeNode` component.
- `part/part.go`: Add new database functions for tree queries.
- `static/custom.js` (maybe): new file for collapse/expand logic if needed. 