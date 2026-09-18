# 🎨 Terminal Prompt Integration Guide

You can display your active `git-user` profile directly in your terminal prompt. The `git-user prompt` command is extremely fast and will only output your profile name if you are currently inside a Git repository.

## 🚀 Easy Automatic Installation (Recommended)

Simply run the interactive installer in your terminal:
```bash
git-user prompt install
```
This command auto-detects your active shell (Zsh, Bash, Fish) or prompt framework (Starship), takes a safe timestamped backup of your config file, and automatically appends/integrates the prompt configuration for you!

---

## 📑 Manual Setup Guide
- [Starship (Cross-Shell)](#-starship-cross-shell)
- [Zsh & Oh My Zsh](#-zsh--oh-my-zsh)
- [Bash](#-bash)
- [Fish Shell](#-fish-shell)
- [Oh My Posh (Cross-Shell)](#-oh-my-posh-cross-shell)
- [Spaceship Prompt (Zsh)](#-spaceship-prompt-zsh)
- [Nushell](#-nushell)
- [Powerlevel10k (Advanced Zsh)](#-powerlevel10k-advanced-zsh)

---

## 🚀 Starship (Cross-Shell)

[Starship](https://starship.rs/) is a popular, cross-shell prompt framework. Adding a custom module for `git-user` is very straightforward.

**Step 1: Append the configuration**
Open your terminal and run the following command. It will safely append the `gituser` custom module to the bottom of your configuration file:
```bash
cat << 'EOF' >> ~/.config/starship.toml

[custom.gituser]
command = "git-user prompt"
when = "git rev-parse --is-inside-work-tree 2>/dev/null"
format = "[$output]($style) "
style = "bold blue"
EOF
```

**Step 2: Verify the installation**
Starship automatically reloads its configuration files on the fly. Simply navigate into any Git repository, and you should instantly see your active profile displayed in bold blue!

*(Optional Manual Setup: If you prefer, you can open `~/.config/starship.toml` in your text editor and manually paste the block from Step 1).*

---

## 🐚 Zsh & Oh My Zsh

For standard Zsh or Oh My Zsh setups, we utilize `RPROMPT` with `add-zsh-hook` and `PROMPT_SUBST` for robust theme compatibility (including Oh My Zsh themes like `robbyrussell`, `agnoster`, etc.).

**Step 1: Inject the prompt function**
Run the following command to safely append the integration function to your `.zshrc` file:
```bash
cat << 'EOF' >> ~/.zshrc

# --- git-user prompt integration ---
setopt PROMPT_SUBST 2>/dev/null

function _git_user_prompt() {
  local user
  user=$(git-user prompt 2>/dev/null)
  if [[ -n "$user" ]]; then
    local icon=" "
    if [[ "$TERM" == "linux" || "$TERM" == "dumb" ]]; then
      icon="git:"
    fi
    if [[ -n "$GIT_USER_PROMPT_ICON" ]]; then
      icon="$GIT_USER_PROMPT_ICON"
    fi
    echo "%F{blue}${icon}${user}%f"
  fi
}

autoload -Uz add-zsh-hook 2>/dev/null

_git_user_setup_prompt() {
  local p="$(_git_user_prompt)"
  if [[ -n "$p" ]]; then
    if [[ -z "$_GIT_USER_ORIG_RPROMPT" && -n "$RPROMPT" && "$RPROMPT" != *"$p"* ]]; then
      _GIT_USER_ORIG_RPROMPT="$RPROMPT"
    fi
    RPROMPT="${p}${_GIT_USER_ORIG_RPROMPT:+ $_GIT_USER_ORIG_RPROMPT}"
  elif [[ -n "$_GIT_USER_ORIG_RPROMPT" ]]; then
    RPROMPT="$_GIT_USER_ORIG_RPROMPT"
  fi
}

if functions add-zsh-hook >/dev/null 2>&1; then
  add-zsh-hook precmd _git_user_setup_prompt
else
  RPROMPT='$(_git_user_prompt)'
fi
EOF
```

**Step 2: Reload Zsh**
Apply the changes immediately by sourcing your configuration file:
```bash
source ~/.zshrc
```

---

## 🐚 Bash

For standard Bash users, the prompt dynamically updates `PS1` using `PROMPT_COMMAND` while preserving your custom prompt colors/theme and properly enclosing escape sequences so cursor positioning and line-wrapping never break.

**Step 1: Inject the prompt function**
Run the following command in your terminal to append the integration to your `.bashrc`:
```bash
cat << 'EOF' >> ~/.bashrc

# --- git-user prompt integration ---
__git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [ -n "$user" ]; then
    local icon=" "
    if [ "$TERM" = "linux" ] || [ "$TERM" = "dumb" ]; then
      icon="git:"
    fi
    if [ -n "$GIT_USER_PROMPT_ICON" ]; then
      icon="$GIT_USER_PROMPT_ICON"
    fi
    printf "\001\033[1;34m\002%s%s\001\033[0m\002 " "$icon" "$user"
  fi
}

__git_user_update_ps1() {
  if [ -z "$__GIT_USER_ORIG_PS1" ]; then
    __GIT_USER_ORIG_PS1="$PS1"
  fi
  local p=$(__git_user_prompt)
  PS1="${p}${__GIT_USER_ORIG_PS1}"
}

if [[ ! "$PROMPT_COMMAND" =~ __git_user_update_ps1 ]]; then
  PROMPT_COMMAND="__git_user_update_ps1${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
fi
EOF
```

**Step 2: Reload Bash**
Apply the changes immediately by sourcing your configuration file:
```bash
source ~/.bashrc
```

---

## 🐟 Fish Shell

Fish handles right-aligned prompts using `fish_right_prompt`. The integration preserves any existing right prompt functions defined by your theme or configuration.

**Step 1: Create the prompt configuration**
Run the following command to install the integration to Fish's autoload directory:
```bash
mkdir -p ~/.config/fish/conf.d
cat << 'EOF' > ~/.config/fish/conf.d/git_user_prompt.fish
# --- git-user prompt integration ---
if status is-interactive
    if functions -q fish_right_prompt; and not functions -q __git_user_orig_right_prompt
        functions -c fish_right_prompt __git_user_orig_right_prompt
    end

    function fish_right_prompt -d "Display active git-user profile in right prompt"
        set -l git_user (git-user prompt 2>/dev/null)
        if test -n "$git_user"
            set -l icon " "
            if test "$TERM" = "linux" -o "$TERM" = "dumb"
                set icon "git:"
            end
            if set -q GIT_USER_PROMPT_ICON
                set icon "$GIT_USER_PROMPT_ICON"
            end
            set_color blue
            echo -n "$icon$git_user"
            set_color normal
        end
        if functions -q __git_user_orig_right_prompt
            echo -n " "
            __git_user_orig_right_prompt
        end
    end
end
EOF
```

**Step 2: Reload Fish**
Simply close and reopen your terminal, or run:
```bash
source ~/.config/fish/conf.d/git_user_prompt.fish
```

---

## 🚀 Oh My Posh (Cross-Shell)

[Oh My Posh](https://ohmyposh.dev/) uses explicit JSON, YAML, or TOML theme files, making it best suited for a quick manual edit.

**Step 1: Open your theme file**
Open your active Oh My Posh theme file in your favorite text editor.

**Step 2: Locate your target block**
Find the `blocks` array and locate the segment block where you want the git-user profile to appear (usually in a block with `"alignment": "right"`).

**Step 3: Insert the custom segment**
Copy and paste this new `command` segment into the `segments` array:
```json
{
  "type": "command",
  "style": "plain",
  "foreground": "blue",
  "properties": {
    "command": "git-user prompt",
    "prefix": " "
  }
}
```

**Step 4: Save and view**
Save the file. Oh My Posh will instantly reload and display the profile!

---

## 🚀 Spaceship Prompt (Zsh)

If you use the [Spaceship Zsh prompt](https://spaceship-prompt.sh/), you can add a custom section specifically for `git-user`.

**Step 1: Inject the Spaceship integration**
Run the following command to append the custom section to your `.zshrc`:
```bash
cat << 'EOF' >> ~/.zshrc

# --- git-user Spaceship integration ---
spaceship_gituser() {
  local user=$(git-user prompt 2>/dev/null)
  [[ -z "$user" ]] && return
  spaceship::section \
    "blue" \
    " " \
    "$user"
}
SPACESHIP_PROMPT_ORDER=(
  $SPACESHIP_PROMPT_ORDER
  gituser
)
EOF
```

**Step 2: Reload Zsh**
Apply the changes immediately:
```bash
source ~/.zshrc
```

---

## 🐚 Nushell

Nushell relies on its `env.nu` file to control the right prompt.

**Step 1: Open your environment config**
Open your `env.nu` configuration file (e.g. `~/.config/nushell/env.nu` on Linux/macOS or `%APPDATA%\nushell\env.nu` on Windows). You can easily find its path by running `config env` inside Nushell.

**Step 2: Update the right prompt**
Append the following block to your `env.nu` file:
```nushell
# --- git-user prompt integration ---
$env.PROMPT_COMMAND_RIGHT = {||
    let user = (do -i { git-user prompt } | complete)
    if ($user.exit_code == 0) and ($user.stdout != "") {
        let u = ($user.stdout | str trim)
        let icon = (if ($env.TERM? == "linux" or $env.TERM? == "dumb") { "git:" } else { " " })
        let icon = (if ($env.GIT_USER_PROMPT_ICON? != null) { $env.GIT_USER_PROMPT_ICON } else { $icon })
        $"(ansi blue)($icon)($u)(ansi reset)"
    } else {
        ""
    }
}
```

**Step 3: Reload Nushell**
Save the file and restart Nushell to see your new right prompt!

---

## 🛠 Powerlevel10k (Advanced Zsh)

Because Powerlevel10k generates a highly complex configuration file (`~/.p10k.zsh`), it cannot be safely edited with automated bash commands. You must add the segment manually.

**Step 1: Open the configuration file**
Open your Powerlevel10k configuration file in a text editor:
```bash
nano ~/.p10k.zsh
```

**Step 2: Register the custom element**
Search for the `POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS` array (usually around line 43) and add `gituser` to the list. It should look like this:
```zsh
  typeset -g POWERLEVEL9K_RIGHT_PROMPT_ELEMENTS=(
    # ... other segments ...
    time
    gituser   # <--- Add this line here
  )
```

**Step 3: Define the custom function**
Scroll **all the way to the bottom** of the file. Right before the very last `}` character, paste this custom function block:
```zsh
  typeset -g POWERLEVEL9K_GITUSER_FOREGROUND=blue
  typeset -g POWERLEVEL9K_GITUSER_VISUAL_IDENTIFIER_EXPANSION=''
  
  function _gituser_cache_update() {
    export _GIT_USER_PROMPT_CACHE=$(git-user prompt 2>/dev/null)
  }
  
  # Register the precmd hook so it only runs once per prompt, never during terminal resize!
  autoload -Uz add-zsh-hook
  add-zsh-hook precmd _gituser_cache_update

  function prompt_gituser() {
    if [[ -n "$_GIT_USER_PROMPT_CACHE" ]]; then
      p10k segment -t "$_GIT_USER_PROMPT_CACHE"
    fi
  }
```

**Step 4: Save and reload**
Save the file (in nano: <kbd>Ctrl+O</kbd>, <kbd>Enter ↵</kbd>, <kbd>Ctrl+X</kbd>) and reload your terminal configuration:
```bash
source ~/.zshrc
```
