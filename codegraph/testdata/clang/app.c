int leaf(void) { return 1; }

int helper(void) { return leaf(); }

int entry(void) { return helper() + 1; }
