# AI Features

## Two Ways To Use AI (Both Free Options Exist)

### Option A - Ollama (100% Free, Runs On Your Machine)

Step by step:

1. Go to https://ollama.com and click Download
2. Install it like any normal app (Windows installer, Mac .dmg, Linux script)
3. Open a terminal and run:

```bash
ollama pull qwen2.5-coder:1.5b
```

This downloads a 4GB AI model once, stored on your machine.

4. Open `gitflow-tui` - AI activates automatically
5. The status bar shows the AI badge with `AI:ollama`

Ollama runs entirely offline after the model downloads. Nothing is sent anywhere. It is completely private.

### Option B - Anthropic API (Free Tier Available)

Step by step:

1. Go to https://console.anthropic.com
2. Create a free account
3. Go to API Keys and create a key
4. Set the key:

Windows:

```powershell
$env:ANTHROPIC_API_KEY="sk-ant-..."
```

Mac/Linux:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

5. Open `gitflow-tui` - the status bar shows the AI badge with `AI:anthropic`

The free tier has usage limits. Ollama is recommended if you plan to use AI features heavily.

## What Each AI Feature Does (Plain English)

### Commit Message Suggestion (ctrl+a in commit prompt)

Say you staged 3 files that add a login form. Instead of typing the commit message yourself, press `ctrl+a`. The AI reads the staged changes and suggests something like:

```text
feat(auth): add email and password login form
```

You can accept it, edit it, or ignore it.

### Merge Conflict Predictor (X key on any branch)

Imagine you are on `feature/payment-api` and want to merge into `develop`. Before running `git merge`, press `X`. The AI runs a silent simulation and tells you:

- Whether it will be clean or conflicting
- Which exact files will conflict and why
- A plain English explanation of what each side is trying to do
- A recommendation for how to resolve it

### Diff Explainer (E key in Diff panel)

If you have a big diff open, press `E` and the AI streams a plain English explanation directly into the panel, word by word as it thinks. That gives you a quick summary without manually parsing hundreds of lines of red and green output.

### Stash Explainer (E key in Stash panel)

If you have something like `stash@{3}` and cannot remember what it was, move to that stash entry and press `E`. The AI reads the stash diff and tells you what you were working on.

## If AI Is Not Available

The tool works exactly the same without AI. The status bar shows no AI badge. Pressing `X`, `E`, or `ctrl+a` shows a hint explaining how to enable it. Nothing breaks. Nothing errors.
