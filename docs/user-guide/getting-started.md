# Getting Started with AI Work Studio

*Your personal AI assistant that learns and adapts to your specific needs*

Welcome to AI Work Studio! This guide will have you up and running in 5 minutes with your first goal and objective.

## Quick Start (5-Minute Setup)

### Step 1: Install AI Work Studio (2 minutes)

**Automated Installation (Recommended):**

```bash
# Linux/macOS - Download and install
curl -sSL https://raw.githubusercontent.com/yourusername/ai-work-studio/main/install.sh | bash

# Windows PowerShell - Run as Administrator
iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/yourusername/ai-work-studio/main/install.ps1'))
```

**Manual Installation:** See our [detailed installation guide](../installation.md) for custom setups.

### Step 2: Initial Configuration (1 minute)

Run the setup wizard:

```bash
ai-work-studio --setup
```

The setup will ask you to choose:
- **LLM Provider**: Start with "local" (no API keys needed) or use "anthropic"/"openai" if you have API keys
- **Data Location**: Default location is fine for most users
- **Budget Limits**: Set reasonable daily limits if using paid APIs ($10-50)

### Step 3: Launch and Create Your First Goal (2 minutes)

Start the AI Work Studio interface:

```bash
ai-work-studio
```

A window will open showing the main interface. Let's create your first goal:

1. **Click "New Goal"** in the Goals panel
2. **Enter a goal title**: "Organize my work files"
3. **Describe your goal**: "I want to organize my documents and projects into a clear, searchable structure that saves me time finding files"
4. **Set success criteria**: "I can find any work document in under 30 seconds"
5. **Click "Create Goal"**

### Step 4: Create Your First Objective

Now let's create a specific objective to work toward your goal:

1. **Select your new goal** from the Goals panel
2. **Click "New Objective"** in the Objectives panel
3. **Enter objective title**: "Analyze current file structure"
4. **Describe the task**: "Review my Documents folder and identify patterns in how files are currently organized"
5. **Set the context**: Point to your Documents folder or wherever you keep work files
6. **Click "Create Objective"**

### Step 5: Execute Your First Objective

1. **Select your objective** and click "Execute"
2. **Watch the magic happen**: The AI will analyze your files and provide insights
3. **Review the results**: You'll see what it discovered about your current organization
4. **Provide feedback**: Let it know what worked and what didn't

🎉 **Congratulations!** You just completed your first AI Work Studio session.

## What Just Happened?

You've experienced AI Work Studio's core learning cycle:

- **You set a goal** (organize work files) with clear success criteria
- **You defined an objective** (analyze current structure) as a step toward that goal
- **The system executed** your objective using its current methods
- **You provided feedback**, helping it learn your preferences

Over time, the system will:
- Learn your organizational preferences
- Develop custom methods for handling your files
- Suggest improvements and optimizations
- Adapt to your workflow patterns

## Next Steps

### Immediate Actions (Next 30 minutes)

1. **Create more objectives** for your file organization goal:
   - "Create a folder structure proposal"
   - "Move files into new structure"
   - "Set up automated file sorting rules"

2. **Create a second goal** in a different area:
   - Personal productivity
   - Learning a new skill
   - Managing email
   - Planning projects

3. **Explore the interface**:
   - Check the Methods panel to see what approaches the system is learning
   - Review the Status panel for system insights
   - Try the different visualization options

### This Week

- **Use it daily**: The more you interact, the better it learns your patterns
- **Create 3-5 goals**: Cover different areas of your work or life
- **Provide feedback**: Tell it what works and what doesn't
- **Adjust settings**: Fine-tune configuration as you learn your preferences

### This Month

- **Watch it evolve**: Notice how methods become more personalized
- **Increase complexity**: Give it more challenging, multi-step objectives
- **Review progress**: Check how well it's meeting your success criteria
- **Backup your data**: Run periodic backups of your goals and methods

## Understanding the Interface

### Main Panels

- **Goals Panel** (Left): Your high-level objectives and success criteria
- **Objectives Panel** (Center): Specific tasks that advance your goals
- **Methods Panel** (Right): The approaches the system is learning to use
- **Status Panel** (Bottom): Real-time feedback and system insights

### Key Concepts

- **Goal**: A high-level outcome you want to achieve (e.g., "Be more organized")
- **Objective**: A specific task that advances a goal (e.g., "Clean up Downloads folder")
- **Method**: An approach the system learns for accomplishing objectives
- **Context**: Information about your situation, preferences, and constraints

## Common First Goals

Here are some popular first goals to inspire you:

### Personal Organization
- "Organize my digital files and photos"
- "Create a personal productivity system"
- "Manage my email effectively"
- "Plan and track personal goals"

### Work & Career
- "Improve my project management workflow"
- "Learn a new professional skill"
- "Organize my research and notes"
- "Automate repetitive work tasks"

### Learning & Development
- "Master a programming language"
- "Stay current with industry trends"
- "Build a personal knowledge base"
- "Develop better study habits"

### Health & Wellness
- "Create a sustainable exercise routine"
- "Improve my sleep schedule"
- "Plan healthy meals for the week"
- "Manage stress and mental health"

## Tips for Success

### Write Clear Goals
- Be specific about what success looks like
- Include measurable outcomes when possible
- Focus on outcomes, not activities
- Start with smaller, achievable goals

### Create Good Objectives
- Break big goals into smaller, actionable steps
- Provide enough context for the system to understand
- Be open to different approaches
- Don't micromanage - let the system explore

### Give Useful Feedback
- Be specific about what worked and what didn't
- Explain your reasoning and preferences
- Highlight concerns or constraints the system should know
- Celebrate successes to reinforce good methods

### Be Patient with Learning
- The system improves with experience
- Early attempts may be clunky - this is normal
- Consistency beats perfection
- Trust the process - it gets smarter over time

## Getting Help

- **Troubleshooting**: See our [troubleshooting guide](troubleshooting.md)
- **Core Concepts**: Learn more in our [concepts guide](concepts.md)
- **Workflows**: Discover patterns in our [workflows guide](workflows.md)
- **Configuration**: Advanced setup in our [configuration guide](configuration.md)

## Quick Reference Commands

```bash
# Launch the GUI
ai-work-studio

# Check system status
ai-work-studio-agent --health-check

# View configuration
ai-work-studio --config-check

# View logs
tail -f ~/.local/share/ai-work-studio/logs/app.log

# Backup data
ai-work-studio --backup

# Get help
ai-work-studio --help
```

## Ready to Go!

You now have everything you need to start benefiting from AI Work Studio. Remember:

- **Start simple**: One goal, one objective
- **Be consistent**: Daily interaction accelerates learning
- **Provide feedback**: Help it understand your preferences
- **Be patient**: The system gets smarter over time

Your AI assistant is ready to learn about you and help you achieve your goals. Let's get started!

---

**Next**: Learn about the [core concepts](concepts.md) that make AI Work Studio unique.