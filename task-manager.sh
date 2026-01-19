#!/bin/bash

# AI Work Studio Task Manager
# Manages task completion tracking and prepares context for Claude Code sessions

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUIDE_FILE="${SCRIPT_DIR}/ai-work-studio-development-guide.md"
PROGRESS_FILE="${SCRIPT_DIR}/.task-progress.json"
TASK_CONTEXT_FILE="${SCRIPT_DIR}/.current-task-context.md"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Initialize progress file if it doesn't exist
init_progress() {
    if [ ! -f "$PROGRESS_FILE" ]; then
        echo '{"completed_tasks": [], "current_task": null}' > "$PROGRESS_FILE"
    fi
}

# Get list of all tasks from the guide
list_all_tasks() {
    grep -n "^### Task [0-9]" "$GUIDE_FILE" | sed 's/^\([0-9]*\):### Task \([^:]*\):\(.*\)/\2|\1|\3/'
}

# Get completed tasks
get_completed_tasks() {
    python3 -c "import json; print('\n'.join(json.load(open('$PROGRESS_FILE'))['completed_tasks']))" 2>/dev/null || echo ""
}

# Get next incomplete task
get_next_task() {
    local completed=$(get_completed_tasks)
    local all_tasks=$(list_all_tasks)

    while IFS='|' read -r task_id line_num title; do
        if ! echo "$completed" | grep -q "^${task_id}$"; then
            echo "${task_id}|${line_num}|${title}"
            return
        fi
    done <<< "$all_tasks"

    echo "NONE|0|All tasks completed!"
}

# Extract task section from guide
extract_task() {
    local task_id="$1"
    local start_line=$(grep -n "^### Task ${task_id}:" "$GUIDE_FILE" | cut -d: -f1)

    if [ -z "$start_line" ]; then
        echo "Error: Task ${task_id} not found in guide"
        return 1
    fi

    # Find the next task or section marker
    local end_line=$(tail -n +$((start_line + 1)) "$GUIDE_FILE" | grep -n "^### Task \|^## Phase" | head -1 | cut -d: -f1)

    if [ -z "$end_line" ]; then
        # No next task found, go to end of file
        end_line=$(wc -l < "$GUIDE_FILE")
    else
        end_line=$((start_line + end_line - 1))
    fi

    # Extract the philosophy and conventions section (first ~200 lines)
    head -n 200 "$GUIDE_FILE" > "$TASK_CONTEXT_FILE"

    echo "" >> "$TASK_CONTEXT_FILE"
    echo "---" >> "$TASK_CONTEXT_FILE"
    echo "" >> "$TASK_CONTEXT_FILE"
    echo "# CURRENT TASK" >> "$TASK_CONTEXT_FILE"
    echo "" >> "$TASK_CONTEXT_FILE"

    # Extract the specific task
    sed -n "${start_line},${end_line}p" "$GUIDE_FILE" >> "$TASK_CONTEXT_FILE"

    echo "$TASK_CONTEXT_FILE"
}

# Mark task as complete
mark_complete() {
    local task_id="$1"
    python3 << EOF
import json
with open('$PROGRESS_FILE', 'r') as f:
    data = json.load(f)
if '$task_id' not in data['completed_tasks']:
    data['completed_tasks'].append('$task_id')
    data['completed_tasks'].sort()
with open('$PROGRESS_FILE', 'w') as f:
    json.dump(data, f, indent=2)
print(f"✓ Marked task $task_id as complete")
EOF
}

# Show status
show_status() {
    echo -e "${BLUE}=== AI Work Studio - Task Progress ===${NC}"
    echo ""

    local completed=$(get_completed_tasks | wc -l)
    local total=$(list_all_tasks | wc -l)

    echo -e "Completed: ${GREEN}${completed}${NC} / ${total} tasks"
    echo ""

    if [ $completed -gt 0 ]; then
        echo -e "${GREEN}Completed Tasks:${NC}"
        get_completed_tasks | while read task; do
            echo "  ✓ Task $task"
        done
        echo ""
    fi

    local next=$(get_next_task)
    local next_id=$(echo "$next" | cut -d'|' -f1)
    local next_title=$(echo "$next" | cut -d'|' -f3)

    if [ "$next_id" != "NONE" ]; then
        echo -e "${YELLOW}Next Task:${NC} $next_id -$next_title"
    else
        echo -e "${GREEN}🎉 All tasks completed!${NC}"
    fi
}

# Prepare for next task
prepare_next() {
    local next=$(get_next_task)
    local next_id=$(echo "$next" | cut -d'|' -f1)
    local next_title=$(echo "$next" | cut -d'|' -f3)

    if [ "$next_id" = "NONE" ]; then
        echo -e "${GREEN}🎉 All tasks completed! No more tasks to do.${NC}"
        return 0
    fi

    echo -e "${BLUE}=== Preparing Task $next_id ===${NC}"
    echo -e "Title:$next_title"
    echo ""

    # Extract task context
    local context_file=$(extract_task "$next_id")

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} Task context prepared: $context_file"
        echo ""
        echo -e "${YELLOW}Instructions for Claude Code:${NC}"
        echo "---"
        echo "You are working on AI Work Studio development."
        echo ""
        echo "1. Read the task context file: $context_file"
        echo "2. This file contains:"
        echo "   - Development philosophy and conventions (apply to ALL code)"
        echo "   - The specific task requirements for Task $next_id"
        echo "3. Complete the task following all conventions"
        echo "4. Run all tests to verify completion"
        echo "5. When done, mark as complete by running:"
        echo "   ./task-manager.sh complete $next_id"
        echo "---"
        echo ""
        echo -e "${BLUE}Copy this command to start:${NC}"
        echo "claude-pro"
        echo ""
        echo -e "${BLUE}Then paste this into Claude:${NC}"
        echo "Read $(realpath $context_file) and complete the task described. Follow all development conventions in the file."
    else
        echo -e "${RED}✗${NC} Failed to prepare task context"
        return 1
    fi
}

# Main command processing
init_progress

case "${1:-status}" in
    status)
        show_status
        ;;
    next)
        prepare_next
        ;;
    complete)
        if [ -z "$2" ]; then
            echo "Error: Please specify task ID (e.g., ./task-manager.sh complete 1.1)"
            exit 1
        fi
        mark_complete "$2"
        echo ""
        show_status
        ;;
    list)
        echo -e "${BLUE}=== All Tasks ===${NC}"
        list_all_tasks | while IFS='|' read -r task_id line_num title; do
            if get_completed_tasks | grep -q "^${task_id}$"; then
                echo -e "  ${GREEN}✓${NC} Task $task_id -$title"
            else
                echo -e "    Task $task_id -$title"
            fi
        done
        ;;
    reset)
        echo -e "${YELLOW}Warning: This will reset all progress!${NC}"
        read -p "Are you sure? (yes/no): " confirm
        if [ "$confirm" = "yes" ]; then
            echo '{"completed_tasks": [], "current_task": null}' > "$PROGRESS_FILE"
            echo -e "${GREEN}✓${NC} Progress reset"
        else
            echo "Cancelled"
        fi
        ;;
    help|*)
        echo "AI Work Studio Task Manager"
        echo ""
        echo "Usage: ./task-manager.sh [command]"
        echo ""
        echo "Commands:"
        echo "  status              Show current progress (default)"
        echo "  next                Prepare context for next task"
        echo "  complete <task_id>  Mark a task as complete (e.g., complete 1.1)"
        echo "  list                List all tasks with completion status"
        echo "  reset               Reset all progress (requires confirmation)"
        echo "  help                Show this help message"
        echo ""
        echo "Typical workflow:"
        echo "  1. ./task-manager.sh next       # Prepare next task"
        echo "  2. claude-pro                   # Start Claude Code"
        echo "  3. [Paste the instructions shown]"
        echo "  4. [Work on the task in Claude Code]"
        echo "  5. ./task-manager.sh complete 1.1   # Mark as done"
        echo "  6. Repeat from step 1"
        ;;
esac
