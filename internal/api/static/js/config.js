// Configuration Page JS
console.log('Config page JS loaded');

// Alpine.js theme manager
function themeManager() {
    return {
        init() {
            const savedTheme = localStorage.getItem('hallmonitor_theme') || 'dark';
            document.documentElement.dataset.theme = savedTheme;
        },
        toggle() {
            const html = document.documentElement;
            const newTheme = html.dataset.theme === 'dark' ? 'light' : 'dark';
            html.dataset.theme = newTheme;
            localStorage.setItem('hallmonitor_theme', newTheme);
        }
    };
}

// Configuration page manager
function configPageManager() {
    return {
        activeTab: 'monitors',
        monitors: [],
        groups: [],
        showMonitorModal: false,
        showGroupModal: false,
        editingMonitor: null,
        editingGroup: null,
        showToast: false,
        toastMessage: '',
        toastType: 'success',
        monitorForm: {
            type: 'http',
            name: '',
            url: '',
            target: '',
            query: '',
            queryType: 'A',
            interval: '30s',
            timeout: '10s',
            expectedStatus: 200,
            enabled: true,
            group: ''
        },
        groupForm: {
            name: '',
            interval: '30s',
            monitors: []
        },
        storageForm: {
            backend: 'badger',
            badger: {
                enabled: true,
                path: './data/hallmonitor.db',
                retentionDays: 30,
                enableAggregation: true
            },
            postgres: {
                host: 'localhost',
                port: 5432,
                database: 'hallmonitor',
                user: 'hallmonitor',
                password: '',
                sslmode: 'disable',
                retentionDays: 30
            },
            influxdb: {
                url: 'http://localhost:8086',
                token: '',
                org: 'hallmonitor',
                bucket: 'monitor_results'
            }
        },

        async init() {
            await this.loadData();
        },

        async loadData() {
            try {
                const response = await fetch('/api/v1/config');
                const data = await response.json();

                // Extract monitors from groups
                this.monitors = [];
                this.groups = data.monitoring.groups || [];

                this.groups.forEach(group => {
                    if (group.monitors) {
                        group.monitors.forEach(monitor => {
                            this.monitors.push({
                                ...monitor,
                                group: group.name,
                                enabled: monitor.enabled === undefined || monitor.enabled === null ? true : monitor.enabled
                            });
                        });
                    }
                });

                // Load storage configuration
                if (data.storage) {
                    this.storageForm.backend = data.storage.backend || 'badger';
                    this.storageForm.badger = {
                        enabled: data.storage.badger?.enabled ?? true,
                        path: data.storage.badger?.path || './data/hallmonitor.db',
                        retentionDays: data.storage.badger?.retentionDays || 30,
                        enableAggregation: data.storage.badger?.enableAggregation ?? true
                    };
                    if (data.storage.postgres) {
                        this.storageForm.postgres = {
                            host: data.storage.postgres.host || 'localhost',
                            port: data.storage.postgres.port || 5432,
                            database: data.storage.postgres.database || 'hallmonitor',
                            user: data.storage.postgres.user || 'hallmonitor',
                            password: data.storage.postgres.password || '',
                            sslmode: data.storage.postgres.sslmode || 'disable',
                            retentionDays: data.storage.postgres.retentionDays || 30
                        };
                    }
                    if (data.storage.influxdb) {
                        this.storageForm.influxdb = {
                            url: data.storage.influxdb.url || 'http://localhost:8086',
                            token: data.storage.influxdb.token || '',
                            org: data.storage.influxdb.org || 'hallmonitor',
                            bucket: data.storage.influxdb.bucket || 'monitor_results'
                        };
                    }
                }
            } catch (error) {
                console.error('Failed to load config:', error);
                this.toast('Failed to load configuration', 'error');
            }
        },

        getEmptyMonitorForm() {
            return {
                type: 'http',
                name: '',
                url: '',
                target: '',
                query: '',
                queryType: 'A',
                interval: '30s',
                timeout: '10s',
                expectedStatus: 200,
                enabled: true,
                group: this.groups.length > 0 ? this.groups[0].name : ''
            };
        },

        getEmptyGroupForm() {
            return {
                name: '',
                interval: '30s',
                monitors: []
            };
        },

        openAddMonitor() {
            this.editingMonitor = null;
            this.monitorForm = this.getEmptyMonitorForm();
            this.showMonitorModal = true;
        },

        openEditMonitor(monitor) {
            this.editingMonitor = monitor;
            this.monitorForm = { ...monitor };
            this.showMonitorModal = true;
        },

        async saveMonitor() {
            try {
                const payload = {
                    type: this.monitorForm.type,
                    name: this.monitorForm.name,
                    interval: this.monitorForm.interval || '30s',
                    timeout: this.monitorForm.timeout || '10s',
                    enabled: this.monitorForm.enabled
                };

                // Add type-specific fields
                if (this.monitorForm.type === 'http') {
                    payload.url = this.monitorForm.url;
                    if (this.monitorForm.expectedStatus) {
                        payload.expectedStatus = parseInt(this.monitorForm.expectedStatus);
                    }
                } else if (this.monitorForm.type === 'tcp' || this.monitorForm.type === 'ping') {
                    payload.target = this.monitorForm.target;
                } else if (this.monitorForm.type === 'dns') {
                    payload.query = this.monitorForm.query;
                    payload.queryType = this.monitorForm.queryType || 'A';
                }

                let response;
                if (this.editingMonitor) {
                    // Update existing monitor
                    response = await fetch(`/api/v1/monitors/${this.editingMonitor.name}`, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ monitor: payload })
                    });
                } else {
                    // Create new monitor
                    response = await fetch('/api/v1/monitors', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            group_name: this.monitorForm.group,
                            monitor: payload
                        })
                    });
                }

                const result = await response.json();

                if (result.success) {
                    this.showMonitorModal = false;
                    await this.loadData();
                    this.toast(result.message, 'success');
                } else {
                    this.toast(result.error || result.message, 'error');
                }
            } catch (error) {
                console.error('Failed to save monitor:', error);
                this.toast('Failed to save monitor', 'error');
            }
        },

        async confirmDeleteMonitor(monitor) {
            if (!confirm(`Delete monitor "${monitor.name}"?`)) return;

            try {
                const response = await fetch(`/api/v1/monitors/${monitor.name}`, {
                    method: 'DELETE'
                });

                const result = await response.json();

                if (result.success) {
                    await this.loadData();
                    this.toast(result.message, 'success');
                } else {
                    this.toast(result.error || result.message, 'error');
                }
            } catch (error) {
                console.error('Failed to delete monitor:', error);
                this.toast('Failed to delete monitor', 'error');
            }
        },

        openAddGroup() {
            this.editingGroup = null;
            this.groupForm = this.getEmptyGroupForm();
            this.showGroupModal = true;
        },

        openEditGroup(group) {
            this.editingGroup = group;
            this.groupForm = { ...group };
            this.showGroupModal = true;
        },

        async saveGroup() {
            try {
                const payload = {
                    name: this.groupForm.name,
                    interval: this.groupForm.interval || '30s',
                    monitors: this.groupForm.monitors || []
                };

                let response;
                if (this.editingGroup) {
                    // Update existing group
                    response = await fetch(`/api/v1/groups/${this.editingGroup.name}`, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ group: payload })
                    });
                } else {
                    // Create new group
                    response = await fetch('/api/v1/groups', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ group: payload })
                    });
                }

                const result = await response.json();

                if (result.success) {
                    this.showGroupModal = false;
                    await this.loadData();
                    this.toast(result.message, 'success');
                } else {
                    this.toast(result.error || result.message, 'error');
                }
            } catch (error) {
                console.error('Failed to save group:', error);
                this.toast('Failed to save group', 'error');
            }
        },

        async confirmDeleteGroup(group) {
            if (!confirm(`Delete group "${group.name}"? This will also delete all monitors in this group.`)) return;

            try {
                const response = await fetch(`/api/v1/groups/${group.name}`, {
                    method: 'DELETE'
                });

                const result = await response.json();

                if (result.success) {
                    await this.loadData();
                    this.toast(result.message, 'success');
                } else {
                    this.toast(result.error || result.message, 'error');
                }
            } catch (error) {
                console.error('Failed to delete group:', error);
                this.toast('Failed to delete group', 'error');
            }
        },

        getMonitorTarget(monitor) {
            if (monitor.url) return monitor.url;
            if (monitor.target) return monitor.target;
            if (monitor.query) return monitor.query;
            return 'N/A';
        },

        formatDuration(duration) {
            if (!duration) return 'N/A';
            if (typeof duration === 'string') return duration;
            // Convert nanoseconds to human-readable format
            const seconds = duration / 1000000000;
            if (seconds < 60) return `${seconds}s`;
            if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
            return `${Math.floor(seconds / 3600)}h`;
        },

        async saveStorageConfig() {
            try {
                // Load current full config
                const configResponse = await fetch('/api/v1/config');
                const currentConfig = await configResponse.json();

                // Update storage section
                currentConfig.storage = {
                    backend: this.storageForm.backend,
                    badger: {
                        enabled: this.storageForm.badger.enabled,
                        path: this.storageForm.badger.path,
                        retentionDays: this.storageForm.badger.retentionDays,
                        enableAggregation: this.storageForm.badger.enableAggregation
                    }
                };

                // Save updated config
                const response = await fetch('/api/v1/config', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ config: currentConfig })
                });

                const result = await response.json();

                if (result.success) {
                    this.toast('Storage configuration saved successfully. Restart required for changes to take effect.', 'success');
                } else {
                    this.toast(result.error || result.message, 'error');
                }
            } catch (error) {
                console.error('Failed to save storage config:', error);
                this.toast('Failed to save storage configuration', 'error');
            }
        },

        toast(message, type = 'success') {
            this.toastMessage = message;
            this.toastType = type;
            this.showToast = true;
            setTimeout(() => {
                this.showToast = false;
            }, 3000);
        }
    };
}
