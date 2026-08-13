export type PromptOptions = {
	title: string;
	message: string;
	warning?: string;
	confirmLabel: string;
	danger?: boolean;
	run: () => void;
};

export type ActivePrompt = {
	title: string;
	message: string;
	warning: string;
	confirmLabel: string;
	danger: boolean;
	run: () => void;
};

export function createPrompt() {
	let active = $state<ActivePrompt | null>(null);

	return {
		get current() {
			return active;
		},
		ask(options: PromptOptions) {
			active = {
				title: options.title,
				message: options.message,
				warning: options.warning ?? '',
				confirmLabel: options.confirmLabel,
				danger: options.danger ?? false,
				run: options.run
			};
		},
		accept() {
			const run = active?.run;
			active = null;
			run?.();
		},
		dismiss() {
			active = null;
		}
	};
}
