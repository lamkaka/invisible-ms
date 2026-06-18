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
            const minutes = h * 60;
            if (minutes < 1) {
                return (minutes * 60).toFixed(1) + 's';
            }
            if (h < 1) {
                return minutes.toFixed(1) + 'm';
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
    // Staff Component
    // ============================================
    Alpine.data('staff', () => ({
        staffList: [],
        availableRoles: [],
        sortKey: null,
        sortAsc: true,
        loading: false,
        error: null,
        showCreateModal: false,
        showEditModal: false,
        saving: false,
        form: {
            staff_id: '',
            name: '',
            phone_number: '',
            assigned_roles: []
        },

        async init() {
            await Promise.all([
                this.fetchStaff(),
                this.fetchRoles()
            ]);
        },

        async fetchRoles() {
            try {
                const response = await fetch('/api/companies/ACME/roles');
                if (!response.ok) return;
                const roles = await response.json();
                this.availableRoles = roles.map(r => r.name);
            } catch (err) {
                console.error('Failed to fetch roles:', err);
            }
        },

        async fetchStaff() {
            this.loading = true;
            this.error = null;

            try {
                const response = await fetch('/api/staff?company_code=ACME');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.staffList = await response.json();
                this.applySort();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to fetch staff:', err);
            } finally {
                this.loading = false;
            }
        },

        sortBy(key) {
            if (this.sortKey === key) {
                this.sortAsc = !this.sortAsc;
            } else {
                this.sortKey = key;
                this.sortAsc = true;
            }
            this.applySort();
        },

        applySort() {
            if (!this.sortKey) return;
            const key = this.sortKey;
            const asc = this.sortAsc;
            this.staffList.sort((a, b) => {
                let va = a[key], vb = b[key];
                if (typeof va === 'string') {
                    va = va.toLowerCase();
                    vb = (vb || '').toLowerCase();
                }
                if (va == null) va = '';
                if (vb == null) vb = '';
                if (va < vb) return asc ? -1 : 1;
                if (va > vb) return asc ? 1 : -1;
                return 0;
            });
        },

        sortIndicator(key) {
            if (this.sortKey !== key) return '';
            return this.sortAsc ? ' ▲' : ' ▼';
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

        editStaff(staffMember) {
            this.form = {
                staff_id: staffMember.staff_id,
                name: staffMember.name,
                phone_number: staffMember.phone_number,
                assigned_roles: [...staffMember.assigned_roles]
            };
            this.showEditModal = true;
        },

        async saveStaff() {
            this.saving = true;

            try {
                const isEdit = !!this.form.staff_id;
                const url = isEdit
                    ? `/api/staff/${this.form.staff_id}`
                    : '/api/staff';

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
                await this.fetchStaff();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to save staff:', err);
            } finally {
                this.saving = false;
            }
        },

        async toggleStatus(staffMember) {
            try {
                const response = await fetch(`/api/staff/${staffMember.staff_id}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        ...staffMember,
                        is_active: !staffMember.is_active
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                await this.fetchStaff();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to toggle staff status:', err);
            }
        },

        closeModal() {
            this.showCreateModal = false;
            this.showEditModal = false;
            this.showDeleteModal = false;
            this.form = {
                staff_id: '',
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
        sortKey: null,
        sortAsc: true,
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
                this.applySort();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to fetch action types:', err);
            } finally {
                this.loading = false;
            }
        },

        sortBy(key) {
            if (this.sortKey === key) {
                this.sortAsc = !this.sortAsc;
            } else {
                this.sortKey = key;
                this.sortAsc = true;
            }
            this.applySort();
        },

        applySort() {
            if (!this.sortKey) return;
            const key = this.sortKey;
            const asc = this.sortAsc;
            this.actionTypes.sort((a, b) => {
                let va = a[key], vb = b[key];
                if (typeof va === 'string') {
                    va = va.toLowerCase();
                    vb = (vb || '').toLowerCase();
                }
                if (va == null) va = '';
                if (vb == null) vb = '';
                if (va < vb) return asc ? -1 : 1;
                if (va > vb) return asc ? 1 : -1;
                return 0;
            });
        },

        sortIndicator(key) {
            if (this.sortKey !== key) return '';
            return this.sortAsc ? ' ▲' : ' ▼';
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

    // ============================================
    // Roles Component
    // ============================================
    Alpine.data('roles', () => ({
        roles: [],
        sortKey: null,
        sortAsc: true,
        loading: false,
        error: null,
        showCreateModal: false,
        showEditModal: false,
        showDeleteModal: false,
        saving: false,
        formError: null,
        companyCode: 'ACME',
        form: {
            name: '',
            hourly_rate: ''
        },

        async init() {
            this.companyCode = this.$el.dataset.companyCode || 'ACME';
            await this.fetchRoles();
        },

        async fetchRoles() {
            this.loading = true;
            this.error = null;

            try {
                const response = await fetch(`/api/companies/${this.companyCode}/roles`);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.roles = await response.json();
                this.applySort();
            } catch (err) {
                this.error = err.message;
                console.error('Failed to fetch roles:', err);
            } finally {
                this.loading = false;
            }
        },

        sortBy(key) {
            if (this.sortKey === key) {
                this.sortAsc = !this.sortAsc;
            } else {
                this.sortKey = key;
                this.sortAsc = true;
            }
            this.applySort();
        },

        applySort() {
            if (!this.sortKey) return;
            const key = this.sortKey;
            const asc = this.sortAsc;
            this.roles.sort((a, b) => {
                let va = a[key], vb = b[key];
                if (typeof va === 'string') {
                    va = va.toLowerCase();
                    vb = (vb || '').toLowerCase();
                }
                if (va == null) va = '';
                if (vb == null) vb = '';
                if (va < vb) return asc ? -1 : 1;
                if (va > vb) return asc ? 1 : -1;
                return 0;
            });
        },

        sortIndicator(key) {
            if (this.sortKey !== key) return '';
            return this.sortAsc ? ' ▲' : ' ▼';
        },

        editRole(role) {
            this.form = {
                name: role.name,
                hourly_rate: role.hourly_rate
            };
            this.showEditModal = true;
        },

        deleteRole(role) {
            this.form = {
                name: role.name,
                hourly_rate: role.hourly_rate
            };
            this.showDeleteModal = true;
        },

        async saveRole() {
            this.saving = true;
            this.formError = null;

            try {
                const isEdit = this.showEditModal;
                const url = isEdit
                    ? `/api/companies/${this.companyCode}/roles/${this.form.name}`
                    : `/api/companies/${this.companyCode}/roles`;

                const method = isEdit ? 'PUT' : 'POST';
                const body = isEdit
                    ? JSON.stringify({ hourly_rate: parseFloat(this.form.hourly_rate) })
                    : JSON.stringify({
                        role_name: this.form.name,
                        hourly_rate: parseFloat(this.form.hourly_rate)
                    });

                const response = await fetch(url, {
                    method,
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || `HTTP error! status: ${response.status}`);
                }

                this.closeModal();
                await this.fetchRoles();
            } catch (err) {
                this.formError = err.message;
                console.error('Failed to save role:', err);
            } finally {
                this.saving = false;
            }
        },

        async confirmDelete() {
            this.saving = true;
            this.formError = null;

            try {
                const response = await fetch(`/api/companies/${this.companyCode}/roles/${this.form.name}`, {
                    method: 'DELETE'
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || `HTTP error! status: ${response.status}`);
                }

                this.closeModal();
                await this.fetchRoles();
            } catch (err) {
                this.formError = err.message;
                console.error('Failed to delete role:', err);
            } finally {
                this.saving = false;
            }
        },

        closeModal() {
            this.showCreateModal = false;
            this.showEditModal = false;
            this.showDeleteModal = false;
            this.form = {
                name: '',
                hourly_rate: ''
            };
            this.formError = null;
        },

        formatCurrency(amount) {
            if (!amount && amount !== 0) return '$0.00';
            return new Intl.NumberFormat('en-US', {
                style: 'currency',
                currency: 'USD'
            }).format(parseFloat(amount));
        }
    }));
});
