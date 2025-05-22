<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import axios from 'axios'
import { message, Statistic, Card, Row, Col } from 'ant-design-vue'
import { SearchOutlined, DatabaseOutlined } from '@ant-design/icons-vue'

defineProps({
  msg: String,
})

const apiMessage = ref('Loading...')
const error = ref(null)
const entries = ref([])
const loading = ref(false)
const searchQuery = ref('')
const totalRecords = ref(0) // Total number of records in the database

// Search parameters for each column
const urlSearch = ref('')
const userSearch = ref('')
const passSearch = ref('')

// Table pagination, sorting, and filtering
const tableState = reactive({
  pagination: {
    current: 1,
    pageSize: 20,
    total: 0,
    showSizeChanger: true,
    pageSizeOptions: ['10', '20', '50', '100', '200'],
    showTotal: (total) => `Total ${total} items`
  },
  sorter: {},
  filters: {},
})

// Table columns definition
const columns = [
  {
    title: 'URL',
    dataIndex: 'url',
    key: 'url',
    sorter: true,
    width: '40%',
    customFilterDropdown: true,
    className: 'wrapped-cell',
    ellipsis: false,
  },
  {
    title: 'Username',
    dataIndex: 'user',
    key: 'user',
    sorter: true,
    width: '30%',
    customFilterDropdown: true,
    className: 'wrapped-cell',
    ellipsis: false,
  },
  {
    title: 'Password',
    dataIndex: 'pass',
    key: 'pass',
    sorter: true,
    width: '30%',
    customFilterDropdown: true,
    className: 'wrapped-cell',
    ellipsis: false,
  },
]

// Helper functions for column filtering
const applyColumnFilter = (columnName, value) => {
  if (columnName === 'url') {
    urlSearch.value = value || '';
  } else if (columnName === 'user') {
    userSearch.value = value || '';
  } else if (columnName === 'pass') {
    passSearch.value = value || '';
  }
  
  // Trigger search with the updated filter
  searchEntries();
}

const clearColumnFilter = (columnName) => {
  if (columnName === 'url') {
    urlSearch.value = '';
  } else if (columnName === 'user') {
    userSearch.value = '';
  } else if (columnName === 'pass') {
    passSearch.value = '';
  }
  
  // Refresh data if needed
  searchEntries();
}

// Last updated timestamp
const lastUpdated = ref(new Date())

// Format the last updated time
const formattedLastUpdated = computed(() => {
  return lastUpdated.value.toLocaleString()
})

// Filtered entries based on search query
const filteredEntries = computed(() => {
  // If no filters are applied, return all entries
  if (!searchQuery.value && !urlSearch.value && !userSearch.value && !passSearch.value) {
    return entries.value
  }
  
  return entries.value.filter(entry => {
    // Apply the general search filter if provided
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      if (!(entry.url.toLowerCase().includes(query) || 
          entry.user.toLowerCase().includes(query) || 
          entry.pass.toLowerCase().includes(query))) {
        return false
      }
    }
    
    // Apply column-specific filters if provided
    if (urlSearch.value && !entry.url.toLowerCase().includes(urlSearch.value.toLowerCase())) {
      return false
    }
    if (userSearch.value && !entry.user.toLowerCase().includes(userSearch.value.toLowerCase())) {
      return false
    }
    if (passSearch.value && !entry.pass.toLowerCase().includes(passSearch.value.toLowerCase())) {
      return false
    }
    
    return true
  })
})

// Fetch total record count from API
const fetchTotalRecords = async () => {
  try {
    const response = await axios.get('/api/stats')
    totalRecords.value = response.data.totalRecords
    // Update the last updated timestamp
    lastUpdated.value = new Date()
  } catch (err) {
    console.error('Failed to fetch total record count:', err)
  }
}

// Fetch data from API
const fetchEntries = async (params = {}) => {
  loading.value = true
  try {
    // Construct pagination parameters for the API
    const queryParams = new URLSearchParams()
    const page = params.page || tableState.pagination.current
    const pageSize = params.pageSize || tableState.pagination.pageSize
    
    queryParams.append('page', page)
    queryParams.append('pageSize', pageSize)
    
    // Add sorting params if available
    if (params.sortField) {
      queryParams.append('sortField', params.sortField)
      queryParams.append('sortOrder', params.sortOrder)
    }
    
    const response = await axios.get(`/api/entries?${queryParams.toString()}`)
    const data = response.data

    // Update entries with the response data
    entries.value = data.items || []
    
    // Update pagination with server-side pagination metadata
    tableState.pagination = {
      ...tableState.pagination,
      current: data.page || tableState.pagination.current,
      pageSize: data.pageSize || tableState.pagination.pageSize,
      total: data.total || 0,
    }

    // Store pagination metadata for UI display
    const paginationMeta = {
      total: data.total || 0,
      current: data.page || 1,
      pageSize: data.pageSize || 10,
      hasNext: data.hasNext || false,
      hasPrevious: data.hasPrevious || false,
      nextPage: data.nextPage || 1,
      prevPage: data.prevPage || 1,
      offset: data.offset || 0,
      totalPages: data.totalPages || 1
    }

    console.log('Pagination metadata:', paginationMeta)
    
    return {
      data: data.items || [],
      total: data.total || 0,
      pagination: paginationMeta
    }
  } catch (err) {
    error.value = 'Failed to fetch entries: ' + err.message
    message.error('Failed to fetch entries')
    console.error('API Error:', err)
  } finally {
    loading.value = false
  }
}

// Search entries from API
const searchEntries = async (value = null) => {
  // Use value parameter if provided (from a-input-search)
  if (value !== null && typeof value === 'string') {
    searchQuery.value = value;
    // Reset to first page when performing a new search with new query
    tableState.pagination.current = 1;
  }
  
  // Refresh the total record count whenever we search
  fetchTotalRecords();
  
  // If no search parameters are provided, fetch all entries
  if (!searchQuery.value && !urlSearch.value && !userSearch.value && !passSearch.value) {
    return await fetchEntries({
      page: tableState.pagination.current,
      pageSize: tableState.pagination.pageSize
    })
  }
  
  loading.value = true
  try {
    // Build the query parameters
    const queryParams = new URLSearchParams()
    
    // Add pagination parameters
    queryParams.append('page', tableState.pagination.current)
    queryParams.append('pageSize', tableState.pagination.pageSize)
    
    if (searchQuery.value) {
      queryParams.append('q', searchQuery.value)
    }
    
    if (urlSearch.value) {
      queryParams.append('url', urlSearch.value)
    }
    
    if (userSearch.value) {
      queryParams.append('user', userSearch.value)
    }
    
    if (passSearch.value) {
      queryParams.append('pass', passSearch.value)
    }
    
    // Add sorting params if available
    if (tableState.sorter && tableState.sorter.field) {
      queryParams.append('sortField', tableState.sorter.field)
      queryParams.append('sortOrder', tableState.sorter.order)
    }
    
    const response = await axios.get(`/api/search?${queryParams.toString()}`)
    const data = response.data
    
    // Update entries with the filtered data
    entries.value = data.items || []
    
    // Update pagination with server-side pagination metadata
    tableState.pagination = {
      ...tableState.pagination,
      current: data.page || tableState.pagination.current,
      pageSize: data.pageSize || tableState.pagination.pageSize,
      total: data.total || 0,
    }
    
    // Store pagination metadata for UI display
    const paginationMeta = {
      total: data.total || 0,
      current: data.page || 1,
      pageSize: data.pageSize || 10,
      hasNext: data.hasNext || false,
      hasPrevious: data.hasPrevious || false,
      nextPage: data.nextPage || 1,
      prevPage: data.prevPage || 1,
      offset: data.offset || 0,
      totalPages: data.totalPages || 1
    }
    
    console.log('Search pagination metadata:', paginationMeta)
    
    return {
      data: data.items || [],
      total: data.total || 0,
      pagination: paginationMeta
    }
  } catch (err) {
    error.value = 'Failed to search entries: ' + err.message
    message.error('Failed to search entries')
    console.error('API Error:', err)
  } finally {
    loading.value = false
  }
}

// Handle table change (pagination, filters, sorter)
const handleTableChange = (pagination, filters, sorter) => {
  tableState.pagination = pagination
  tableState.filters = filters
  tableState.sorter = sorter
  
  // Prepare params for server-side pagination and sorting
  const params = {
    page: pagination.current,
    pageSize: pagination.pageSize,
  }
  
  // Add sorting params if available
  if (sorter && sorter.field) {
    params.sortField = sorter.field
    params.sortOrder = sorter.order
  }
  
  // Re-fetch data with new pagination parameters
  if (searchQuery.value || urlSearch.value || userSearch.value || passSearch.value) {
    // If we have active search filters, use the search endpoint
    searchEntries()
  } else {
    // Otherwise use the entries endpoint
    fetchEntries(params)
  }
}

onMounted(async () => {
  try {
    // Fetch welcome message
    const response = await axios.get('/api/hello')
    apiMessage.value = response.data
    
    // Set initial pagination settings
    tableState.pagination = {
      ...tableState.pagination,
      current: 1,
      pageSize: 10 // Match the pageSize in your table state definition
    }
    
    // Fetch initial data with pagination
    const result = await fetchEntries({
      page: tableState.pagination.current,
      pageSize: tableState.pagination.pageSize
    })
    
    // Total records count will be updated by fetchEntries via API response
    
    // Fetch total record count for the stats display
    await fetchTotalRecords()
  } catch (err) {
    error.value = 'Failed to fetch from API: ' + err.message
    message.error('Failed to load data')
    console.error('API Error:', err)
  }
})
</script>

<template>

  <div class="data-table-container">
    <h2>DataLeak from Telegram</h2>
    
    <!-- Stats Section -->
    <div class="stats-section">
      <a-row>
        <a-col :span="24">
          <a-card>
            <a-statistic 
              title="Total Records in Database" 
              :value="totalRecords" 
              :loading="loading"
            >
              <template #prefix>
                <database-outlined />
              </template>
              <template #suffix>
                <span>records</span>
              </template>
            </a-statistic>
            <div style="margin-top: 8px; font-size: 12px; color: rgba(0,0,0,0.45);">
              Last updated: {{ formattedLastUpdated }}
            </div>
          </a-card>
        </a-col>
      </a-row>
    </div>

    <!-- Search Section -->
    <div class="search-section">
      <!-- Global Search Bar -->
      <a-input
        v-model:value="searchQuery"
        placeholder="Global search across all columns..."
        style="width: 100%; margin-bottom: 16px;"
        allow-clear
        @pressEnter="searchEntries"
      >
      </a-input>

      <!-- Column-specific Search -->
      <div class="column-search-container">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-input
              v-model:value="urlSearch"
              placeholder="Search URL..."
              allow-clear
              @pressEnter="searchEntries"
            />
          </a-col>
          <a-col :span="8">
            <a-input
              v-model:value="userSearch"
              placeholder="Search Username..."
              allow-clear
              @pressEnter="searchEntries"
            />
          </a-col>
          <a-col :span="8">
            <a-input
              v-model:value="passSearch"
              placeholder="Search Password..."
              allow-clear
              @pressEnter="searchEntries"
            />
          </a-col>
        </a-row>
        <a-button
          type="primary"
          style="margin-top: 16px; width: 100%;"
          :loading="loading"
          @click="searchEntries()"
        >
          <search-outlined /> Search
        </a-button>
      </div>
    </div>

    <!-- Ant Design Table -->
    <div v-if="error" class="error">{{ error }}</div>
    <div v-else-if="loading" class="loading">Loading...</div>
    <div v-else-if="!entries.length" class="no-data">No data available</div>
    
    <a-table
      :columns="columns"
      :data-source="entries"
      :pagination="tableState.pagination"
      :loading="loading"
      @change="handleTableChange"
      row-key="id"
      bordered
      class="data-table"
    >
      <!-- URL Column -->
      <template #bodyCell="{ column, text, record }">
        <template v-if="column.key === 'url'">
          <a :href="record.url" target="_blank" rel="noopener noreferrer" style="word-break: break-all; display: inline-block; max-width: 100%;">{{ text }}</a>
        </template>
      </template>
      
      <!-- Custom Filter Dropdowns -->
      <template #customFilterDropdown="{ setSelectedKeys, selectedKeys, confirm, clearFilters, column }">
        <div style="padding: 8px">
          <a-input
            :placeholder="`Search ${column.title}`"
            :value="selectedKeys[0]"
            style="width: 188px; margin-bottom: 8px; display: block;"
            @change="e => setSelectedKeys(e.target.value ? [e.target.value] : [])"
            @pressEnter="() => { 
              confirm(); 
              applyColumnFilter(column.dataIndex, selectedKeys[0]);
            }"
          />
          <a-button
            type="primary"
            size="small"
            style="width: 90px; margin-right: 8px"
            @click="() => { 
              confirm(); 
              applyColumnFilter(column.dataIndex, selectedKeys[0]);
            }"
          >
            Search
          </a-button>
          <a-button
            size="small"
            style="width: 90px"
            @click="() => { 
              clearFilters(); 
              clearColumnFilter(column.dataIndex);
            }"
          >
            Reset
          </a-button>
        </div>
      </template>
    </a-table>
  </div>
</template>

<style scoped>
@import '../assets/styles/home.css';
</style>
