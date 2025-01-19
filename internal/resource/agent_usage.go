package resource

// describe each agent in 18 words or less
var AgentDesc = map[string]string{
	"ask":  "Provide concise, clear answers on various topics",
	"sql":  "Generate queries, enhancing productivity for all user levels.",
	"oh":   "Aid greenfield projects, refactoring, and debugging.",
	"seek": "Conduct web research and gather timely information to create factual, cited reports.",
	"git":  "Generate effective commit messages for projects.",
	"code": "Enable collaborative code editing with LLMs.",
	"pr":   "Create pull request descriptions for local changes.",
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
	The Pull Request (PR) agent is a valuable tool for software developers, streamlining the process of creating pull request descriptions. By leveraging AI capabilities, it generates detailed and informative descriptions for local changes, enhancing communication and collaboration among team members. This tool ensures that pull requests are well-documented, facilitating code reviews and improving the overall efficiency of the development workflow.
`,
}
