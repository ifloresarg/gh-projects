local M = {}

-- Default configuration
M.config = {
  binary = "gh projects",
  width = 0.8,
  height = 0.8,
  border = "rounded",
}

-- Setup function to configure the plugin
function M.setup(opts)
  opts = opts or {}
  M.config = vim.tbl_extend("force", M.config, opts)

  -- Register the :GhProjects command
  vim.api.nvim_create_user_command("GhProjects", function(cmd_info)
    M.open_projects(cmd_info.args)
  end, {
    nargs = "*",
    desc = "Open gh-projects TUI in a floating window",
  })
end

-- Extract owner from git remote URL
local function get_repo_owner()
  local ok, output = pcall(vim.fn.system, "git remote get-url origin")
  if not ok or output:match("^fatal") then
    return nil
  end

  -- Remove trailing newline
  output = output:gsub("\n$", "")

  -- Match HTTPS: https://github.com/owner/repo.git
  local owner_https = output:match("github%.com[:/]([^/]+)/")
  if owner_https then
    return owner_https
  end

  -- Match SSH: git@github.com:owner/repo.git
  local owner_ssh = output:match("git@github%.com:([^/]+)/")
  if owner_ssh then
    return owner_ssh
  end

  return nil
end

-- Open the projects TUI in a floating window
function M.open_projects(args)
  -- Calculate window dimensions
  local vim_width = vim.o.columns
  local vim_height = vim.o.lines
  local win_width = math.floor(vim_width * M.config.width)
  local win_height = math.floor((vim_height - 1) * M.config.height)
  local row = math.floor((vim_height - win_height) / 2)
  local col = math.floor((vim_width - win_width) / 2)

  -- Create a new buffer for the terminal
  local buf = vim.api.nvim_create_buf(false, true)

  -- Open floating window
  local win = vim.api.nvim_open_win(buf, true, {
    relative = "editor",
    width = win_width,
    height = win_height,
    row = row,
    col = col,
    border = M.config.border,
  })

  -- Set buffer options
  vim.bo[buf].bufhidden = "wipe"
  vim.bo[buf].buflisted = false

  -- Build command
  local cmd = M.config.binary
  
  if args and args ~= "" then
    -- User provided args (owner and/or number)
    local parts = vim.split(args, "%s+")
    if #parts >= 2 then
      -- Both owner and number provided
      cmd = cmd .. " --owner " .. parts[1] .. " --number " .. parts[2]
    else
      -- Just owner provided
      cmd = cmd .. " --owner " .. parts[1]
    end
  else
    -- Auto-detect owner from git remote
    local owner = get_repo_owner()
    if owner then
      cmd = cmd .. " --owner " .. owner
    end
  end

  -- Open terminal and start the TUI
  vim.fn.termopen(cmd)

  -- Buffer-local keybindings to close the window
  local opts_map = { noremap = true, silent = true, buffer = buf }
  vim.keymap.set("n", "q", function()
    vim.api.nvim_win_close(win, true)
  end, opts_map)
  vim.keymap.set("n", "<Esc>", function()
    vim.api.nvim_win_close(win, true)
  end, opts_map)

  -- Enter insert mode for the terminal
  vim.cmd("startinsert")
end

return M
