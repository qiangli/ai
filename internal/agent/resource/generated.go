// DO NOT EDIT. This file is generated.
package resource

import _ "embed"

//go:embed prompts/agent_meta_system_role.md
var agent_meta_system_role string

//go:embed prompts/agent_sub_system_role.md
var agent_sub_system_role string

//go:embed prompts/chdir_system_role.md
var chdir_system_role string

//go:embed prompts/doc_compose_system_role.md
var doc_compose_system_role string

//go:embed prompts/docker_input_user_role.md
var docker_input_user_role string

//go:embed prompts/eval_system_role.md
var eval_system_role string

//go:embed prompts/git_message_long.md
var git_message_long string

//go:embed prompts/git_message_short.md
var git_message_short string

//go:embed prompts/git_sub_system_role.md
var git_sub_system_role string

//go:embed prompts/gptr_sub_system_role.md
var gptr_sub_system_role string

//go:embed prompts/image_param_system_role.md
var image_param_system_role string

//go:embed prompts/pr_changelog_format.md
var pr_changelog_format string

//go:embed prompts/pr_changelog_system_role.md
var pr_changelog_system_role string

//go:embed prompts/pr_describe_example.json
var pr_describe_example string

//go:embed prompts/pr_describe_format.md
var pr_describe_format string

//go:embed prompts/pr_describe_schema.json
var pr_describe_schema string

//go:embed prompts/pr_describe_system_role.md
var pr_describe_system_role string

//go:embed prompts/pr_improve_example.json
var pr_improve_example string

//go:embed prompts/pr_improve_format.md
var pr_improve_format string

//go:embed prompts/pr_improve_schema.json
var pr_improve_schema string

//go:embed prompts/pr_improve_system_role.md
var pr_improve_system_role string

//go:embed prompts/pr_review_example.json
var pr_review_example string

//go:embed prompts/pr_review_format.md
var pr_review_format string

//go:embed prompts/pr_review_schema.json
var pr_review_schema string

//go:embed prompts/pr_review_system_role.md
var pr_review_system_role string

//go:embed prompts/pr_sub_system_role.md
var pr_sub_system_role string

//go:embed prompts/pr_user_role.md
var pr_user_role string

//go:embed prompts/script_system_role.md
var script_system_role string

//go:embed prompts/script_user_role.md
var script_user_role string

//go:embed prompts/sql_system_role.md
var sql_system_role string

//go:embed prompts/workspace_system_role.md
var workspace_system_role string

//go:embed prompts/workspace_user_role.md
var workspace_user_role string

var Prompts = map[string]string{
	"agent_meta_system_role":   agent_meta_system_role,
	"agent_sub_system_role":    agent_sub_system_role,
	"chdir_system_role":        chdir_system_role,
	"doc_compose_system_role":  doc_compose_system_role,
	"docker_input_user_role":   docker_input_user_role,
	"eval_system_role":         eval_system_role,
	"git_message_long":         git_message_long,
	"git_message_short":        git_message_short,
	"git_sub_system_role":      git_sub_system_role,
	"gptr_sub_system_role":     gptr_sub_system_role,
	"image_param_system_role":  image_param_system_role,
	"pr_changelog_format":      pr_changelog_format,
	"pr_changelog_system_role": pr_changelog_system_role,
	"pr_describe_example":      pr_describe_example,
	"pr_describe_format":       pr_describe_format,
	"pr_describe_schema":       pr_describe_schema,
	"pr_describe_system_role":  pr_describe_system_role,
	"pr_improve_example":       pr_improve_example,
	"pr_improve_format":        pr_improve_format,
	"pr_improve_schema":        pr_improve_schema,
	"pr_improve_system_role":   pr_improve_system_role,
	"pr_review_example":        pr_review_example,
	"pr_review_format":         pr_review_format,
	"pr_review_schema":         pr_review_schema,
	"pr_review_system_role":    pr_review_system_role,
	"pr_sub_system_role":       pr_sub_system_role,
	"pr_user_role":             pr_user_role,
	"script_system_role":       script_system_role,
	"script_user_role":         script_user_role,
	"sql_system_role":          sql_system_role,
	"workspace_system_role":    workspace_system_role,
	"workspace_user_role":      workspace_user_role,
}

//go:embed common.yaml
var CommonData []byte

type AgentConfig struct {
	Name        string
	Description string
	Overview    string
	Internal    bool
	Data        []byte
}

//go:embed agent/agent.yaml
var agent_agent_yaml_data []byte

//go:embed aider/agent.yaml
var aider_agent_yaml_data []byte

//go:embed ask/agent.yaml
var ask_agent_yaml_data []byte

//go:embed chdir/agent.yaml
var chdir_agent_yaml_data []byte

//go:embed doc/agent.yaml
var doc_agent_yaml_data []byte

//go:embed draw/agent.yaml
var draw_agent_yaml_data []byte

//go:embed eval/agent.yaml
var eval_agent_yaml_data []byte

//go:embed git/agent.yaml
var git_agent_yaml_data []byte

//go:embed gptr/agent.yaml
var gptr_agent_yaml_data []byte

//go:embed oh/agent.yaml
var oh_agent_yaml_data []byte

//go:embed pr/agent.yaml
var pr_agent_yaml_data []byte

//go:embed script/agent.yaml
var script_agent_yaml_data []byte

//go:embed sql/agent.yaml
var sql_agent_yaml_data []byte

//go:embed workspace/agent.yaml
var workspace_agent_yaml_data []byte

var AgentCommandMap = map[string]AgentConfig{
	"agent": {
		Name:        "agent",
		Description: "Dispatch to the most appropriate agent based on the user's input.",
		Internal:    true,
		Data:        agent_agent_yaml_data,
		Overview:    "",
	},
	"aider": {
		Name:        "aider",
		Description: "Integrate LLMs for collaborative coding, refactoring, bug fixing, and test development.",
		Internal:    false,
		Data:        aider_agent_yaml_data,
		Overview:    "The Code agent is an innovative tool that boosts collaborative software development by seamlessly integrating Large Language Models (LLMs) into local environments. By harnessing the power of these models, it can generate new code, refactor existing code, fix bugs, and develop test cases efficiently. This enhances productivity and code quality, making teamwork more efficient and effective.",
	},
	"ask": {
		Name:        "ask",
		Description: "Deliver concise, reliable answers on a wide range of topics.",
		Internal:    false,
		Data:        ask_agent_yaml_data,
		Overview:    "This agent serves as an all-encompassing Q&A platform, enabling users to explore and inquire about a diverse array of topics, from scientific phenomena to cultural practices and technology. Its primary objective is to deliver reliable, relevant, and accessible answers that are both informative and concise. Ideal for learners, educators, and the inquisitive, it offers an invaluable tool for broadening knowledge and swiftly addressing everyday questions. By providing clear and explanatory answers, this agent supports users in their quest for understanding across a wide range of subjects.",
	},
	"chdir": {
		Name:        "chdir",
		Description: "Evaluate users input and locate the directory on the local system the user intends to change to.",
		Internal:    false,
		Data:        chdir_agent_yaml_data,
		Overview:    "",
	},
	"doc": {
		Name:        "doc",
		Description: "Create a polished document by integrating draft materials into the provided template.",
		Internal:    false,
		Data:        doc_agent_yaml_data,
		Overview:    "This advanced AI document agent helps you create polished and coherent documents effortlessly. By integrating your draft materials into a provided example template, the AI ensures the final document adheres strictly to the template's structure and formatting. It pays close attention to headings, subheadings, bullet points, numbering, and overall organization. The AI maintains a consistent writing style that matches the template's tone and formality, and it refines the content for clarity, coherence, and readability. Finally, it conducts a thorough review to ensure the document meets all specified requirements. This agent streamlines the document creation process, delivering high-quality results with minimal effort.",
	},
	"draw": {
		Name:        "draw",
		Description: "Generate images based on user input, providing visual representations of text-based descriptions.",
		Internal:    false,
		Data:        draw_agent_yaml_data,
		Overview:    "This agent is a powerful tool that generates images based on user input, providing visual representations of text-based descriptions. By converting textual descriptions into visual diagrams, charts, or illustrations, the agent helps users communicate complex ideas, processes, or concepts more effectively. It supports various input formats, including natural language descriptions, code snippets, or data tables, and produces corresponding visual outputs that enhance understanding and clarity. Whether you need to create flowcharts, network diagrams, or architectural blueprints, this agent simplifies the image creation process, enabling users to convey information visually with ease.",
	},
	"eval": {
		Name:        "eval",
		Description: "Evaluate and test tools.",
		Internal:    false,
		Data:        eval_agent_yaml_data,
		Overview:    "",
	},
	"git": {
		Name:        "git",
		Description: "Generate git commit message based on users input and the provided diffs.",
		Internal:    true,
		Data:        git_agent_yaml_data,
		Overview:    "",
	},
	"git/long": {
		Name:        "git/long",
		Description: "Generate git commit messages based on the provided diffs using the Conventional Commits specification",
		Internal:    false,
		Data:        git_agent_yaml_data,
		Overview:    "",
	},
	"git/short": {
		Name:        "git/short",
		Description: "Generate concise, one-line git commit messages based on the provided diffs.",
		Internal:    false,
		Data:        git_agent_yaml_data,
		Overview:    "",
	},
	"gptr": {
		Name:        "gptr",
		Description: "Deliver live, realtime, accurate, relevant insights from diverse online sources.",
		Internal:    false,
		Data:        gptr_agent_yaml_data,
		Overview:    "The agent is a comprehensive tool designed for advanced digital exploration and research. It serves users seeking insights from diverse online sources, whether for academic research, competitive business analysis, or trend monitoring. Leveraging search technologies, it efficiently navigates the internet to provide accurate, timely, and contextually relevant information. With autonomous capabilities, it creates detailed, factual reports complete with citations, tackling issues like misinformation. Customizable and user-friendly, the tool aggregates data from over multiple sources, making digital content access and analysis reliable and efficient.",
	},
	"oh": {
		Name:        "oh",
		Description: "Engineering assistant promoting incremental development and detailed refactoring support.",
		Internal:    false,
		Data:        oh_agent_yaml_data,
		Overview:    "OpenHands is an engineering assistant tool that simplifies tasks by encouraging an incremental approach. Start with basic exercises like creating a 'hello world' script and progressively improve your project. It's ideal for greenfield projects, allowing you to begin with simple tasks such as developing a React TODO app and gradually adding features. OpenHands effectively integrates new code into existing systems and supports step-by-step code refactoring. For troubleshooting, detailed context is essential. The best outcomes are achieved by approaching tasks in small steps, providing specific details, sharing context, and making frequent commits.",
	},
	"pr": {
		Name:        "pr",
		Description: "Enhance PR management with automated summaries, reviews, suggestions, and changelog updates.",
		Internal:    true,
		Data:        pr_agent_yaml_data,
		Overview:    "The PR agent is a robust tool designed to optimize pull request management by automating several key tasks. It generates detailed and accurate summaries, titles, and labels for PR descriptions, reducing manual effort. With its review functionality, developers receive tailored feedback on potential issues, security vulnerabilities, and the overall review process, ensuring high-quality code. Additionally, it offers code suggestions to improve existing code within the PR. The changelog feature further boosts productivity by automatically updating the CHANGELOG.md file, capturing all relevant changes.",
	},
	"pr/changelog": {
		Name:        "pr/changelog",
		Description: "Update the CHANGELOG.md file with the PR changes",
		Internal:    false,
		Data:        pr_agent_yaml_data,
		Overview:    "",
	},
	"pr/describe": {
		Name:        "pr/describe",
		Description: "Generate PR description - title, type, summary, code walkthrough and labels",
		Internal:    false,
		Data:        pr_agent_yaml_data,
		Overview:    "",
	},
	"pr/improve": {
		Name:        "pr/improve",
		Description: "Provide code suggestions for improving the PR",
		Internal:    false,
		Data:        pr_agent_yaml_data,
		Overview:    "",
	},
	"pr/review": {
		Name:        "pr/review",
		Description: "Give feedback about the PR, possible issues, security concerns, review effort and more",
		Internal:    false,
		Data:        pr_agent_yaml_data,
		Overview:    "",
	},
	"script": {
		Name:        "script",
		Description: "Assist with scripting, command execution, and troubleshooting shell tasks.",
		Internal:    false,
		Data:        script_agent_yaml_data,
		Overview:    "The Script agent is a versatile tool that assists users in executing system commands, creating shell scripts, and troubleshooting various scripting tasks. By providing guidance on command syntax, script structure, and error handling, it simplifies the process of writing and executing scripts. Whether you are a novice or an experienced script writer, this agent offers valuable support in automating tasks, managing system configurations, and troubleshooting common scripting issues. It serves as a reliable companion for script development and execution, enhancing productivity and efficiency in system administration and automation.",
	},
	"sql": {
		Name:        "sql",
		Description: "Streamline SQL query generation, helping users derive insights without SQL expertise.",
		Internal:    false,
		Data:        sql_agent_yaml_data,
		Overview:    "This SQL agent streamlines query generation, enabling users to derive insights from databases by asking questions without requiring SQL expertise. It integrates with databases like PostgreSQL, ensuring security and efficiency through privacy and improved accuracy with continuous training. Designed for both enthusiasts and professionals, the tool assists in writing, modifying, and optimizing SQL queries, offering guidance on a wide range of tasks from schema management to complex query execution. By simplifying SQL interactions, it accelerates query development, aids debugging, and boosts productivity, serving both beginners and expert database administrators alike.",
	},
	"workspace": {
		Name:        "workspace",
		Description: "Determines the user's workspace based on user's input.",
		Internal:    false,
		Data:        workspace_agent_yaml_data,
		Overview:    "",
	},
}
