# 🎨 Terminal Prompt Integration Guide

You can display your active `git-user` profile directly in your terminal prompt. The `git-user prompt` command is extremely fast and will only output your profile name if you are currently inside a Git repository.

Because modifying terminal configuration files automatically can be risky, we leave the integration up to you. Follow the detailed step-by-step instructions below for your specific shell or prompt framework.

## 📑 Table of Contents
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

For standard Zsh or Oh My Zsh setups, we can utilize the built-in `RPROMPT` (Right Prompt) feature.

**Step 1: Inject the prompt function**
Run the following command to safely append the integration function to your `.zshrc` file:
```bash
cat << 'EOF' >> ~/.zshrc

# --- git-user prompt integration ---
function _git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [[ -n "$user" ]]; then
    echo "%F{blue} ${user}%f"
  fi
}
RPROMPT='$(_git_user_prompt)'
EOF
```

**Step 2: Reload Zsh**
Apply the changes immediately by sourcing your configuration file:
```bash
source ~/.zshrc
```

---

## 🐚 Bash

For standard Bash users, you can prepend the active profile to your `PS1` prompt variable using `PROMPT_COMMAND`.

**Step 1: Inject the prompt function**
Run the following command in your terminal to append the integration to your `.bashrc`:
```bash
cat << 'EOF' >> ~/.bashrc

# --- git-user prompt integration ---
__git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [ -n "$user" ]; then
    echo -e "\033[1;34m ${user}\033[0m "
  fi
}
# Prepend to PS1 dynamically
PROMPT_COMMAND='PS1="$(__git_user_prompt)\u@\h:\w\$ "'
EOF
```

**Step 2: Reload Bash**
Apply the changes immediately by sourcing your configuration file:
```bash
source ~/.bashrc
```

---

## 🐟 Fish Shell

Fish handles right-aligned prompts using a dedicated `fish_right_prompt` function. 

**Step 1: Prepare the functions directory**
Ensure your custom functions directory exists:
```bash
mkdir -p ~/.config/fish/functions
```

**Step 2: Create the prompt file**
Run the following command to create (or overwrite) the right prompt function file:
```bash
cat << 'EOF' > ~/.config/fish/functions/fish_right_prompt.fish
function fish_right_prompt
  set -l git_user (git-user prompt 2>/dev/null)
  if test -n "$git_user"
    set_color blue
    echo -n " $git_user"
    set_color normal
  end
end
EOF
```
*Note: If you already have a heavily customized `fish_right_prompt.fish`, you may need to open it manually and merge the `git_user` logic inside your existing function.*

**Step 3: Reload Fish**
Simply close and reopen your terminal, or type `fish` to start a new session with the updated prompt.

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
Open your `env.nu` configuration file. You can easily find its path by running `config env` inside Nushell.

**Step 2: Update the right prompt**
Find or create the `PROMPT_COMMAND_RIGHT` variable, and add the `git-user prompt` command to it:
```nushell
let-env PROMPT_COMMAND_RIGHT = {||
    let user = (git-user prompt | complete)
    if ($user.exit_code == 0) and ($user.stdout != "") {
        $"(ansi blue) ($user.stdout | str trim)(ansi reset)"
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
Save the file (in nano: `Ctrl+O`, `Enter`, `Ctrl+X`) and reload your terminal configuration:
```bash
source ~/.zshrc
```
