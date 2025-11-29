# CLAUDE.md

## YOU MUST

- Answers should be in Japanese.
- TODOs should always include creating a branch, testing the implementation, committing, pushing, and creating a PR (if not already done)
- Whenever you modify a feature, be sure to modify (or create) a test and make sure the test passes!

## Repository Configuration

- **Author**: FrontWorksDev
- **Repository Name**: image-compressor
- **MCP GitHub API**: always use `image-compressor` repository

## Be sure to do this at the beginning and end of the work when adding a modification feature. Be sure to include everything in the todo each time

- The following operations must be performed at the start of work
  - **At the start**: Always create a dedicated branch (feature/<feature name>, fixed/<fix description>, etc.)
  - **NEVER work directly on the main branch**: Never commit any changes directly to the main branch.
- Be sure to perform the following at the end of the work
  1. create commit

## Git and Collaboration Guidelines

### Commit Messages

- Use Japanese for commit messages (日本語でコミットメッセージを書く)
- Keep title under 50 characters (タイトルは50文字以内)
- Include purpose and expected impact (変更の目的と期待される影響を含める)
- Use natural line breaks, not `\n` (改行に`\n`は使用せず通常の改行)
- Include main changes and feature additions when applicable:
  - **主な変更点：**
  - **機能追加**

### Pull Requests

- Only create PRs when explicitly requested (指示があるまでPRは作成しない)
- Use `gh` command for PR creation
- Include these sections in PR description:
  - **概要：** (Overview) - purpose and background
  - **変更内容：** (Changes) - specific modifications
  - **影響範囲：** (Impact) - effects on other parts
  - **テスト結果：** (Test Results) - testing status
- Add `Closes #**` if branch name contains `issue_**`
- Use markdown formatting where possible

