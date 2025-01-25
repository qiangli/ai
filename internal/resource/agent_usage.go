package resource

// describe each agent in 16 words
var AgentDesc = map[string]string{
	"ask":    "All-encompassing Q&A platform providing concise, reliable answers on diverse topics.",
	"sql":    "Streamlines SQL query generation, helping users derive insights without SQL expertise.",
	"oh":     "Engineering assistant promoting incremental development and detailed refactoring support.",
	"seek":   "Digital exploration tool delivering accurate, relevant insights from diverse online sources.",
	"git":    "Automates Git commit message creation for clarity and consistency in version control.",
	"code":   "Integrates LLMs for collaborative coding, refactoring, bug fixing, and test development.",
	"pr":     "Enhances PR management with automated summaries, reviews, suggestions, and changelog updates.",
	"script": "Receive assistance to execute system commands, create and troubleshoot various shell scripts.",
	"doc":    "Create a polished document by integrating draft materials into the provided template.",
}

// summarize each agent in 100 words
var AgentInfo = map[string]string{
	"ask": `
	This agent serves as an all-encompassing Q&A platform, enabling users to explore and inquire about a diverse array of topics, from scientific phenomena to cultural practices and technology. Its primary objective is to deliver reliable, relevant, and accessible answers that are both informative and concise. Ideal for learners, educators, and the inquisitive, it offers an invaluable tool for broadening knowledge and swiftly addressing everyday questions. By providing clear and explanatory answers, this agent supports users in their quest for understanding across a wide range of subjects.
`,
	"sql": `
	This SQL agent streamlines query generation, enabling users to derive insights from databases by asking questions without requiring SQL expertise. It integrates with databases like PostgreSQL, ensuring security and efficiency through privacy and improved accuracy with continuous training. Designed for both enthusiasts and professionals, the tool assists in writing, modifying, and optimizing SQL queries, offering guidance on a wide range of tasks from schema management to complex query execution. By simplifying SQL interactions, it accelerates query development, aids debugging, and boosts productivity, serving both beginners and expert database administrators alike.
`,
	"oh": `
    OpenHands is an engineering assistant tool that simplifies tasks by encouraging an incremental approach. Start with basic exercises like creating a "hello world" script and progressively improve your project. It's ideal for greenfield projects, allowing you to begin with simple tasks such as developing a React TODO app and gradually adding features. OpenHands effectively integrates new code into existing systems and supports step-by-step code refactoring. For troubleshooting, detailed context is essential. The best outcomes are achieved by approaching tasks in small steps, providing specific details, sharing context, and making frequent commits.
`,
	"seek": `
    The agent is a comprehensive tool designed for advanced digital exploration and research. It serves users seeking insights from diverse online sources, whether for academic research, competitive business analysis, or trend monitoring. Leveraging search technologies, it efficiently navigates the internet to provide accurate, timely, and contextually relevant information. With autonomous capabilities, it creates detailed, factual reports complete with citations, tackling issues like misinformation. Customizable and user-friendly, the tool aggregates data from over multiple sources, making digital content access and analysis reliable and efficient.
`,
	"git": `
    A Git agent is a tool used in software development to aid in creating and managing Git commit messages. It automates and enhances the process, ensuring consistency and clarity in the project's change history. By producing well-structured commit messages, a Git agent helps maintain organized records, facilitates team collaboration, and improves overall efficiency in version control management.
`,
	"code": `
	The Code agent is an innovative tool that boosts collaborative software development by seamlessly integrating Large Language Models (LLMs) into local environments. By harnessing the power of these models, it can generate new code, refactor existing code, fix bugs, and develop test cases efficiently. This enhances productivity and code quality, making teamwork more efficient and effective.
`,
	"pr": `
    The PR agent is a robust tool designed to optimize pull request management by automating several key tasks. It generates detailed and accurate summaries, titles, and labels for PR descriptions, reducing manual effort. With its review functionality, developers receive tailored feedback on potential issues, security vulnerabilities, and the overall review process, ensuring high-quality code. Additionally, it offers code suggestions to improve existing code within the PR. The changelog feature further boosts productivity by automatically updating the CHANGELOG.md file, capturing all relevant changes.
`,
	"script": `
	The Script agent is a versatile tool that assists users in executing system commands, creating shell scripts, and troubleshooting various scripting tasks. By providing guidance on command syntax, script structure, and error handling, it simplifies the process of writing and executing scripts. Whether you are a novice or an experienced script writer, this agent offers valuable support in automating tasks, managing system configurations, and troubleshooting common scripting issues. It serves as a reliable companion for script development and execution, enhancing productivity and efficiency in system administration and automation.
	
`,
	"doc": `
	This advanced AI document agent helps you create polished and coherent documents effortlessly. By integrating your draft materials into a provided example template, the AI ensures the final document adheres strictly to the template's structure and formatting. It pays close attention to headings, subheadings, bullet points, numbering, and overall organization. The AI maintains a consistent writing style that matches the template's tone and formality, and it refines the content for clarity, coherence, and readability. Finally, it conducts a thorough review to ensure the document meets all specified requirements. This agent streamlines the document creation process, delivering high-quality results with minimal effort.
`,
}

var AgentCommands = map[string]string{
	"git": `
  /short:        Generate a short commit message for Git based on the provided information
  /conventional: Generate a commit message for Git based on the provided information according to the Conventional Commits specification at https://www.conventionalcommits.org/en/v1.0.0/#summary
`,
	"pr": `
  /describe:  Generate PR description - title, type, summary, code walkthrough and labels
  /review:    Feedback about the PR, possible issues, security concerns, review effort and more
  /improve:   Code suggestions for improving the PR
  /changelog: Update the CHANGELOG.md file with the PR changes
`,
	"script": `
  Run "ai list-commands" tool to get the complete list of system commands available in the path.
`,
}
