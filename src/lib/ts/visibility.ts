export function whenVisible<Result>(task: () => Result): () => Result | undefined {
	return () => {
		if (typeof document !== 'undefined' && document.hidden) return undefined;
		return task();
	};
}
