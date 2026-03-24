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
	end

	-- Open terminal and start the TUI
	vim.fn.jobstart(cmd, {
		term = true,
		on_exit = function()
			vim.schedule(function()
				if vim.api.nvim_win_is_valid(win) then
					vim.api.nvim_win_close(win, true)
				end
			end)
		end,
	})

	-- Enter insert mode for the terminal
	vim.cmd("startinsert")
end

return M
