#!/bin/bash
set -e

# Check if required tools are installed
check_dependencies() {
    if ! command -v git-cliff &> /dev/null; then
        echo "Error: git-cliff is not installed"
        echo "Install it with: cargo install git-cliff"
        echo "Or download from: https://github.com/orhun/git-cliff/releases"
        exit 1
    fi
}

# Get current version from git tags
get_current_version() {
    CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    echo "Current version: $CURRENT_VERSION"
}

# Calculate new version based on bump type
calculate_new_version() {
    local bump_type="$1"
    local current_num="${CURRENT_VERSION#v}"
    
    case "$bump_type" in
        patch)
            NEW_VERSION=$(echo $current_num | awk -F. '{printf "v%d.%d.%d", $1, $2, $3+1}')
            ;;
        minor)
            NEW_VERSION=$(echo $current_num | awk -F. '{printf "v%d.%d.%d", $1, $2+1, 0}')
            ;;
        major)
            NEW_VERSION=$(echo $current_num | awk -F. '{printf "v%d.%d.%d", $1+1, 0, 0}')
            ;;
        *)
            echo "Usage: $0 {patch|minor|major}"
            exit 1
            ;;
    esac
    
    echo "New version: $NEW_VERSION"
}

# Generate CHANGELOG.md using git-cliff
generate_changelog() {
    echo "Generating CHANGELOG.md..."
    
    # Check if CHANGELOG.md exists
    if [ ! -f "CHANGELOG.md" ]; then
        echo "CHANGELOG.md doesn't exist, creating a full changelog..."
        if git-cliff --output CHANGELOG.md --tag "$NEW_VERSION"; then
            echo "[OK] CHANGELOG.md created successfully"
        else
            echo "[ERROR] Failed to create CHANGELOG.md"
            exit 1
        fi
    else
        echo "CHANGELOG.md exists, prepending new release..."
        # Generate only the new release section and prepend to existing changelog
        if git-cliff --unreleased --tag "$NEW_VERSION" --prepend CHANGELOG.md; then
            echo "[OK] CHANGELOG.md updated successfully"
        else
            echo "[ERROR] Failed to update CHANGELOG.md"
            exit 1
        fi
    fi
}

# Check if CHANGELOG.md has changes
check_changelog_changes() {
    if git diff-index --quiet HEAD -- CHANGELOG.md; then
        echo "No changes to CHANGELOG.md"
        return 1
    else
        echo "CHANGELOG.md has been updated"
        echo "Changes to be committed:"
        git diff --no-color CHANGELOG.md | head -20
        echo ""
        return 0
    fi
}

# Commit CHANGELOG.md if there are changes
commit_changelog() {
    if check_changelog_changes; then
        echo "Committing CHANGELOG.md..."
        git add CHANGELOG.md
        git commit -m "chore: update changelog for $NEW_VERSION"
        echo "[OK] CHANGELOG.md committed"
    fi
}

# Create and push git tag
create_and_push_tag() {
    echo "Creating tag $NEW_VERSION..."
    git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"
    
    echo "Pushing tag and commits..."
    git push origin HEAD
    git push origin "$NEW_VERSION"
    
    echo "[OK] Tag $NEW_VERSION created and pushed!"
    echo "Release workflow should start automatically on GitHub"
}

# Restore CHANGELOG.md on cancellation
restore_changelog() {
    echo "[CANCELLED] Restoring CHANGELOG.md..."
    git checkout -- CHANGELOG.md
    echo "To restore CHANGELOG.md manually: git checkout -- CHANGELOG.md"
}

# Get user confirmation
confirm_release() {
    read -p "Create tag $NEW_VERSION and commit CHANGELOG.md? (y/N): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        return 0
    else
        return 1
    fi
}

# Main function
main() {
    local bump_type="$1"
    
    check_dependencies
    get_current_version
    calculate_new_version "$bump_type"
    generate_changelog
    
    if confirm_release; then
        commit_changelog
        create_and_push_tag
    else
        restore_changelog
    fi
}

# Run main function with all arguments
main "$@"