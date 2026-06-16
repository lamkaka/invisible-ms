// IMS - Alpine.js Data Stores

document.addEventListener('alpine:init', () => {
    // ============================================
    // Dashboard Component
    // ============================================
    Alpine.data('dashboard', () => ({
        stats: null,
        loading: false,
        error: null,
        lastUpdated: null,
        refreshInterval: null,
        maxCost: 0,

        async init() {
            await this.fetchStats();
            // Auto-refresh every 30 seconds
            this.refreshInterval = setInterval(() => this.fetchStats(), 30000);
        },

        destroy() {
            if (this.refreshInterval) {
                clearInterval(this.refreshInterval);
            }
        },

        async fetchStats() {
            this.loading = true;
            this.error = null;

            try {
                const response = await fetch('/api/dashboard/stats?company_code=ACME');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.stats = await response.json();
                this.lastUpdated = new Date().toISOString();

                // Calculate max cost for progress bar scaling
                const costs = Object.values(this.stats?.cost_tracking?.cost_by_role ?? {});
                this.maxCost = Math.max(...costs, 1);
            } catch (err) {
                this.error = err.message;
                console.error('Failed to fetch dashboard stats:', err);
            } finally {
                this.loading = false;
            }
        },

        async refresh() {
            await this.fetchStats();
        },

        formatHours(hours) {
            if (!hours && hours !== 0) return '0.00';
            const h = parseFloat(hours);
            if (h < 1) {
                return (h * 60).toFixed(0) + 'm';
            }
            return h.toFixed(2) + 'h';
        },

        formatCurrency(amount) {
            if (!amount && amount !== 0) return '$0.00';
            return new Intl.NumberFormat('en-US', {
                style: 'currency',
                currency: 'USD'
            }).format(parseFloat(amount));
        },

        formatTime(timestamp) {
            if (!timestamp) return '-';
            try {
                return new Date(timestamp).toLocaleTimeString('en-US', {
                    hour: '2-digit',
                    minute: '2-digit'
                });
            } catch {
                return timestamp;
            }
        },

        getCostPercentage(cost) {
            if (!this.maxCost) return 0;
            return Math.min((parseFloat(cost) / this.maxCost) * 100, 100);
        }
    }));

    // ============================================
    // Workers Component
    // ============================================
    Alpine.data('workers', () => ({
        workers: [],
        loading: false,
        error: null,
        showCreateModal: false,
        showEditModal: false,
        saving: false,
        form: {
            worker_id: '',
            name: '',
            phone_number: '',
            assigned_roles: []
        },

        async init() {
            await this.fetchWorkers();
        },

        async fetchWorkers() {
            this.loading = true;
            this.error = null;

            try {
                const response = await fetch('/api/workers?company_code=ACME');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.workers = await response.json();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to fetch workers:', err);
            } finally {
                this.loading = false;
            }
        },

        getInitials(name) {
            if (!name) return '?';
            return name
                .split(' ')
                .map(part => part[0])
                .join('')
                .toUpperCase()
                .substring(0, 2);
        },

        editWorker(worker) {
            this.form = {
                worker_id: worker.worker_id,
                name: worker.name,
                phone_number: worker.phone_number,
                assigned_roles: [...worker.assigned_roles]
            };
            this.showEditModal = true;
        },

        async saveWorker() {
            this.saving = true;

            try {
                const isEdit = !!this.form.worker_id;
                const url = isEdit
                    ? `/api/workers/${this.form.worker_id}`
                    : '/api/workers';

                const method = isEdit ? 'PUT' : 'POST';

                const response = await fetch(url, {
                    method,
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        ...this.form,
                        company_code: 'ACME'
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                this.closeModal();
                await this.fetchWorkers();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to save worker:', err);
            } finally {
                this.saving = false;
            }
        },

        async toggleStatus(worker) {
            try {
                const response = await fetch(`/api/workers/${worker.worker_id}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        ...worker,
                        is_active: !worker.is_active
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                await this.fetchWorkers();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to toggle worker status:', err);
            }
        },

        closeModal() {
            this.showCreateModal = false;
            this.showEditModal = false;
            this.showDeleteModal = false;
            this.form = {
                worker_id: '',
                name: '',
                phone_number: '',
                assigned_roles: []
            };
            this.formError = null;
        }
    }));

    // ============================================
    // Action Types Component
    // ============================================
    Alpine.data('actionTypes', () => ({
        actionTypes: [],
        loading: false,
        error: null,
        showCreateModal: false,
        showEditModal: false,
        showDeleteModal: false,
        saving: false,
        formError: null,
        companyCode: 'ACME', // default, overridden from data attribute
        form: {
            action_type: '',
            keyword: ''
        },

        async init() {
            this.companyCode = this.$el.dataset.companyCode || 'ACME';
            await this.fetchActionTypes();
        },

        async fetchActionTypes() {
            this.loading = true;
            this.error = null;

            try {
                const response = await fetch(`/api/companies/${this.companyCode}/action-types`);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.actionTypes = await response.json();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to fetch action types:', err);
            } finally {
                this.loading = false;
            }
        },

        editActionType(actionType) {
            this.form = {
                action_type: actionType.action_type,
                keyword: actionType.keyword
            };
            this.showEditModal = true;
        },

        async saveActionType() {
            this.saving = true;
            this.formError = null;

            try {
                const isEdit = this.showEditModal;
                const url = isEdit
                    ? `/api/companies/${this.companyCode}/action-types/${this.form.action_type}`
                    : `/api/companies/${this.companyCode}/action-types`;

                const method = isEdit ? 'PUT' : 'POST';

                const response = await fetch(url, {
                    method,
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        action_type: this.form.action_type,
                        keyword: this.form.keyword
                    })
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || `HTTP error! status: ${response.status}`);
                }

                this.closeModal();
                await this.fetchActionTypes();
            } catch (err) {
                this.formError = err.message;
                console.error('Failed to save action type:', err);
            } finally {
                this.saving = false;
            }
        },

        deleteActionType(actionType) {
            this.form = {
                action_type: actionType.action_type,
                keyword: actionType.keyword
            };
            this.showDeleteModal = true;
        },

        async confirmDelete() {
            this.saving = true;
            this.formError = null;

            try {
                const response = await fetch(`/api/companies/${this.companyCode}/action-types/${this.form.action_type}`, {
                    method: 'DELETE'
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || `HTTP error! status: ${response.status}`);
                }

                this.closeModal();
                await this.fetchActionTypes();
            } catch (err) {
                this.formError = err.message;
                console.error('Failed to delete action type:', err);
            } finally {
                this.saving = false;
            }
        },

        closeModal() {
            this.showCreateModal = false;
            this.showEditModal = false;
            this.showDeleteModal = false;
            this.form = {
                action_type: '',
                keyword: ''
            };
            this.formError = null;
        }
    }));
});
