import React, { useState, useEffect, useRef } from 'react';
import { Bot, Loader2, Terminal, Sparkles, Play, Eye, Settings, Zap, CheckCircle, Clock, AlertCircle, Command, X, Bug, MapPin, Folder, Monitor, HardDrive } from 'lucide-react';
import JellyLoader from './components/jelly-loader.tsx';
import './styles.css';
import WSALogo from '../public/WSA.svg';

function App() {
    const [goal, setGoal] = useState('');
    const [useVision, setUseVision] = useState(false);
    const [output, setOutput] = useState('');
    const [isExecuting, setIsExecuting] = useState(false);
    const [logs, setLogs] = useState([]);
    const [windowSize, setWindowSize] = useState({ width: window.innerWidth, height: window.innerHeight });
    const [selectedModel, setSelectedModel] = useState('llama3.2');
    const [availableModels, setAvailableModels] = useState([]);
    const [executionStage, setExecutionStage] = useState('');
    const [currentCommand, setCurrentCommand] = useState('');
    const [commandHistory, setCommandHistory] = useState([]);
    const [progress, setProgress] = useState(0);
    
    // New state variables
    const [showDebugPanel, setShowDebugPanel] = useState(false);
    const [isOnboarding, setIsOnboarding] = useState(false);
    const [systemContext, setSystemContext] = useState(null);
    const [onboardingStep, setOnboardingStep] = useState(0);
    const [systemMapping, setSystemMapping] = useState({});
    const [liveCommands, setLiveCommands] = useState([]);
    const abortControllerRef = useRef(null);

    // Mapping (onboarding) progress state
    const [isMapping, setIsMapping] = useState(false);
    const [mappingStage, setMappingStage] = useState('');
    const [mappingProgress, setMappingProgress] = useState(0);
    const [showContextModal, setShowContextModal] = useState(false);

    // Debug log for onboarding state
    useEffect(() => {
        console.log('Onboarding state:', isOnboarding, 'System context:', !!systemContext);
    }, [isOnboarding, systemContext]);

    useEffect(() => {
        const handleResize = () => {
            setWindowSize({ width: window.innerWidth, height: window.innerHeight });
        };

        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, []);

    useEffect(() => {
        // Check if system has been mapped before
        const storedContext = localStorage.getItem('wsa-system-context');
        if (storedContext) {
            setSystemContext(JSON.parse(storedContext));
        } else {
            setIsOnboarding(true);
        }

        // Fetch available models on component mount
        fetch('http://localhost:8080/models')
            .then(response => response.json())
            .then(data => {
                setAvailableModels(data.models || []);
            })
            .catch(error => {
                console.error('Failed to fetch models:', error);
                // Fallback to default models
                setAvailableModels(['llama3.2', 'gemma3:12b', 'gpt-oss:20b']);
            });
    }, []);

    const handleExecute = async () => {
        if (!goal.trim()) {
            return;
        }

        console.log('Starting execution with goal:', goal, 'using model:', selectedModel);
        setIsExecuting(true);
        setOutput('');
        setLogs([]);
        setCommandHistory([]);
        setLiveCommands([]);
        setProgress(0);
        setExecutionStage('Analyzing task...');
        setCurrentCommand('');

        // Create abort controller
        abortControllerRef.current = new AbortController();

        setExecutionStage('Analyzing task requirements...');
        setProgress(10);

        // Enhanced payload with system context
        const payload = {
            goal,
            useVision,
            model: selectedModel,
            systemContext: systemContext,
            timestamp: new Date().toISOString()
        };

        fetch('http://localhost:8080/execute', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(payload),
            signal: abortControllerRef.current.signal
        })
            .then((response) => {
                console.log('Response status:', response.status);
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.json();
            })
            .then((data) => {
                console.log('Response data:', data);
                setOutput(data.message);
                setLogs(data.logs || []);
                setCommandHistory(data.logs || []);
                
                // Add backend live commands if available
                if (data.liveCommands && data.liveCommands.length > 0) {
                    const backendCommands = data.liveCommands.map(cmd => ({
                        command: cmd.command,
                        status: cmd.status,
                        timestamp: new Date(parseInt(cmd.timestamp) * 1000).toISOString()
                    }));
                    
                    setLiveCommands(prev => [...prev, ...backendCommands]);
                } else if (data.logs && data.logs.length > 0) {
                    // Fallback to logs if no live commands
                    const backendCommands = data.logs.map(log => ({
                        command: log,
                        status: 'completed',
                        timestamp: new Date().toISOString()
                    }));
                    
                    setLiveCommands(prev => [...prev, ...backendCommands]);
                }
                
                setProgress(100);
                setExecutionStage('Completed successfully');
                setIsExecuting(false);
            })
            .catch((error) => {
                console.error('Fetch error:', error);
                setOutput(`Error: ${error.message}`);
                setExecutionStage('Execution failed');
                setProgress(0);
                setIsExecuting(false);
            });
    };

    const handleKeyPress = (e) => {
        if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
            handleExecute();
        }
    };

    const handleAbort = () => {
        if (abortControllerRef.current) {
            abortControllerRef.current.abort();
            
            // Mark any running commands as failed
            setLiveCommands(prev => 
                prev.map(cmd => 
                    cmd.status === 'running' 
                        ? { ...cmd, status: 'failed' }
                        : cmd
                )
            );
            
            // Add abort command to live commands
            const abortCommand = {
                command: 'ABORT: Task cancelled by user',
                status: 'failed',
                timestamp: new Date().toISOString()
            };
            setLiveCommands(prev => [...prev, abortCommand]);
            
            setIsExecuting(false);
            setExecutionStage('Task aborted by user');
            setProgress(0);
        }
    };

    const mapSystemContext = async () => {
        setExecutionStage('Mapping system context...');
        
        try {
            // Get basic system info from browser
            const basicInfo = {
                platform: navigator.platform,
                userAgent: navigator.userAgent,
                language: navigator.language,
                languages: navigator.languages,
                cookieEnabled: navigator.cookieEnabled,
                onLine: navigator.onLine,
                hardwareConcurrency: navigator.hardwareConcurrency,
                maxTouchPoints: navigator.maxTouchPoints,
                screenWidth: window.screen.width,
                screenHeight: window.screen.height,
                colorDepth: window.screen.colorDepth,
                pixelDepth: window.screen.pixelDepth,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                timestamp: new Date().toISOString()
            };

            // Try to get additional system info from backend
            let backendData = {};
            try {
                const response = await fetch('http://localhost:8080/map-system', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ basicInfo })
                });
                
                if (response.ok) {
                    backendData = await response.json();
                }
            } catch (backendError) {
                console.warn('Backend system mapping unavailable, using client-side mapping:', backendError);
            }

            // Create comprehensive system context
            const context = {
                // Basic system info
                system: {
                    os: basicInfo.platform,
                    userAgent: basicInfo.userAgent,
                    language: basicInfo.language,
                    languages: basicInfo.languages,
                    timezone: basicInfo.timezone,
                    screen: {
                        width: basicInfo.screenWidth,
                        height: basicInfo.screenHeight,
                        colorDepth: basicInfo.colorDepth,
                        pixelDepth: basicInfo.pixelDepth
                    },
                    hardware: {
                        concurrency: basicInfo.hardwareConcurrency,
                        maxTouchPoints: basicInfo.maxTouchPoints
                    }
                },
                
                // Directory structure (from backend if available)
                directories: backendData.directories || {
                    home: backendData.homeDir || '~',
                    documents: backendData.documentsDir || '~/Documents',
                    downloads: backendData.downloadsDir || '~/Downloads',
                    desktop: backendData.desktopDir || '~/Desktop',
                    pictures: backendData.picturesDir || '~/Pictures',
                    music: backendData.musicDir || '~/Music',
                    videos: backendData.videosDir || '~/Videos'
                },
                
                // Installed applications (from backend if available)
                applications: backendData.applications || [],
                
                // System processes (from backend if available)
                processes: backendData.processes || [],
                
                // Environment variables (from backend if available)
                environment: backendData.environment || {},
                
                // Network interfaces (from backend if available)
                network: backendData.network || {},
                
                // File system info (from backend if available)
                filesystem: backendData.filesystem || {},
                
                // Metadata
                mappingDate: new Date().toISOString(),
                version: '1.0.0',
                source: backendData.directories ? 'backend' : 'client-only'
            };
            
            setSystemContext(context);
            localStorage.setItem('wsa-system-context', JSON.stringify(context));
            setIsOnboarding(false);
            
            console.log('System mapping completed:', context);
            
        } catch (error) {
            console.error('System mapping failed:', error);
            
            // Create minimal fallback context
            const fallbackContext = {
                system: {
                    os: navigator.platform,
                    userAgent: navigator.userAgent,
                    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
                },
                directories: {
                    home: '~',
                    documents: '~/Documents',
                    downloads: '~/Downloads',
                    desktop: '~/Desktop'
                },
                mappingDate: new Date().toISOString(),
                version: '1.0.0',
                source: 'fallback'
            };
            
            setSystemContext(fallbackContext);
            localStorage.setItem('wsa-system-context', JSON.stringify(fallbackContext));
            setIsOnboarding(false);
        }
    };

    const startOnboarding = () => {
        console.log('Starting onboarding...');
        // Clear any existing system context to force fresh onboarding
        localStorage.removeItem('wsa-system-context');
        setSystemContext(null);
        setIsOnboarding(true);
        setOnboardingStep(0);
    };

    const completeOnboarding = async () => {
        // Progressive mapping with backend step endpoints
        setLiveCommands([]);
        setIsMapping(true);
        setMappingStage('Starting system mapping...');
        setMappingProgress(5);

        const steps = [
            { url: 'http://localhost:8080/map-system/directories', label: 'Mapping directories' },
            { url: 'http://localhost:8080/map-system/applications', label: 'Scanning applications' },
            { url: 'http://localhost:8080/map-system/processes', label: 'Listing processes' },
            { url: 'http://localhost:8080/map-system/environment', label: 'Collecting environment' },
            { url: 'http://localhost:8080/map-system/network', label: 'Inspecting network' },
            { url: 'http://localhost:8080/map-system/filesystem', label: 'Scanning filesystem' }
        ];

        const collected = {};
        for (let i = 0; i < steps.length; i++) {
            const step = steps[i];
            setMappingStage(step.label + '...');
            setMappingProgress(5 + Math.floor(((i + 1) / steps.length) * 85));

            try {
                const res = await fetch(step.url, { method: 'POST' });
                if (res.ok) {
                    const data = await res.json();
                    // merge according to step
                    if (data.directories) collected.directories = data.directories;
                    if (data.homeDir) collected.homeDir = data.homeDir;
                    if (data.documentsDir) collected.documentsDir = data.documentsDir;
                    if (data.downloadsDir) collected.downloadsDir = data.downloadsDir;
                    if (data.desktopDir) collected.desktopDir = data.desktopDir;
                    if (data.picturesDir) collected.picturesDir = data.picturesDir;
                    if (data.musicDir) collected.musicDir = data.musicDir;
                    if (data.videosDir) collected.videosDir = data.videosDir;
                    if (data.applications) collected.applications = data.applications;
                    if (data.processes) collected.processes = data.processes;
                    if (data.environment) collected.environment = data.environment;
                    if (data.network) collected.network = data.network;
                    if (data.filesystem) collected.filesystem = data.filesystem;

                    // push command to live log
                    if (data.command) {
                        const entry = { command: data.command, status: 'completed', timestamp: new Date().toISOString() };
                        setLiveCommands(prev => [...prev, entry]);
                    }
                } else {
                    const entry = { command: `${step.label} (backend unavailable)`, status: 'failed', timestamp: new Date().toISOString() };
                    setLiveCommands(prev => [...prev, entry]);
                }
            } catch (e) {
                const entry = { command: `${step.label} (error: ${e.message})`, status: 'failed', timestamp: new Date().toISOString() };
                setLiveCommands(prev => [...prev, entry]);
            }
        }

        // Fallback to single-shot mapping to fill any gaps
        await mapSystemContext();

        // Merge with any stored context
        const stored = localStorage.getItem('wsa-system-context');
        const base = stored ? JSON.parse(stored) : {};
        const merged = {
            ...base,
            directories: collected.directories || base.directories,
            applications: collected.applications || base.applications,
            processes: collected.processes || base.processes,
            environment: collected.environment || base.environment,
            network: collected.network || base.network,
            filesystem: collected.filesystem || base.filesystem,
        };
        localStorage.setItem('wsa-system-context', JSON.stringify(merged));
        setSystemContext(merged);
        setMappingProgress(100);
        setMappingStage('System mapping completed');
        setIsMapping(false);
        setIsOnboarding(false);
    };

    const handleModelChange = async (newModel) => {
        if (newModel === selectedModel) return;
        
        try {
            // Unload current model
            await fetch('http://localhost:8080/unload-model', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ model: selectedModel })
            });
            
            // Load new model
            await fetch('http://localhost:8080/load-model', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ model: newModel })
            });
            
            setSelectedModel(newModel);
        } catch (error) {
            console.error('Model switching failed:', error);
            // Still update the UI even if backend fails
            setSelectedModel(newModel);
        }
    };

    const isCompact = windowSize.width < 1000 || windowSize.height < 700;
    const hasOutput = output || isExecuting;

    // Onboarding Modal Component
    const OnboardingModal = () => {
        const handleStartMapping = async () => {
            setIsMapping(true);
            await completeOnboarding();
        };

        return (
            <div className="onboarding-overlay">
                <div className="onboarding-modal">
                    <div className="onboarding-header">
                        <div className="onboarding-icon-container">
                            <MapPin className="onboarding-icon" />
                        </div>
                        <h2>Welcome to Cypher</h2>
                        <p>Let's map your system to provide better automation</p>
                    </div>
                    
                    <div className="onboarding-content">
                        <div className="onboarding-steps">
                            <div className="onboarding-step">
                                <div className="step-icon-container">
                                    <Monitor className="step-icon" />
                                </div>
                                <div className="step-content">
                                    <h3>System Analysis</h3>
                                    <p>Analyze your system structure, installed applications, and common directories for accurate task automation.</p>
                                </div>
                            </div>
                            
                            <div className="onboarding-step">
                                <div className="step-icon-container">
                                    <Folder className="step-icon" />
                                </div>
                                <div className="step-content">
                                    <h3>Directory Mapping</h3>
                                    <p>Map important folders like Documents, Downloads, Desktop, and project directories.</p>
                                </div>
                            </div>
                            
                            <div className="onboarding-step">
                                <div className="step-icon-container">
                                    <HardDrive className="step-icon" />
                                </div>
                                <div className="step-content">
                                    <h3>Local Context Storage</h3>
                                    <p>Store system context locally for faster and more personalized automation suggestions.</p>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div className="onboarding-actions">
                        <button 
                            onClick={() => setIsOnboarding(false)} 
                            className="skip-button"
                            disabled={isMapping}
                        >
                            Skip for now
                        </button>
                        <button 
                            onClick={handleStartMapping} 
                            className="start-mapping-button"
                            disabled={isMapping}
                        >
                            {isMapping ? (
                                <>
                                    <Loader2 className="button-icon spinning" />
                                    Mapping System...
                                </>
                            ) : (
                                <>
                                    <MapPin className="button-icon" />
                                    Start System Mapping
                                </>
                            )}
                        </button>
                    </div>

                    {isMapping && (
                        <div className="onboarding-progress">
                            <div className="progress-section">
                                <div className="progress-header">
                                    <div className="progress-status" style={{ display: 'flex', alignItems: 'center', gap: 12, minHeight: 56 }}>
                                        <JellyLoader numberOfCubes={6} width={120} height={36} cubeWidth={30} cubeHeight={20} dx={10} dy={-5} style={{ marginTop: 6 }} />
                                        <span className="progress-text">{mappingStage}</span>
                                    </div>
                                    <span className="progress-percentage">{mappingProgress}%</span>
                                </div>
                                <div className="progress-bar">
                                    <div 
                                        className="progress-fill" 
                                        style={{ width: `${mappingProgress}%` }}
                                    ></div>
                                </div>
                            </div>

                            {liveCommands.length > 0 && (
                                <div className="live-commands">
                                    <h3>Mapping Steps</h3>
                                    <div className="commands-list">
                                        {liveCommands.map((cmd, index) => (
                                            <div key={index} className={`command-item ${cmd.status}`}>
                                                <CheckCircle className="command-status-icon" />
                                                <span className="command-number">{index + 1}</span>
                                                <code className="command-code">{cmd.command}</code>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        );
    };

    const SystemContextModal = () => {
        const copyContextToClipboard = async () => {
            try {
                await navigator.clipboard.writeText(JSON.stringify(systemContext || {}, null, 2));
            } catch (e) {
                console.warn('Failed to copy context:', e);
            }
        };

        const handleRemap = () => {
            setShowContextModal(false);
            localStorage.removeItem('wsa-system-context');
            setSystemContext(null);
            setIsOnboarding(true);
            setOnboardingStep(0);
        };

        return (
            <div className="onboarding-overlay">
                <div className="onboarding-modal">
                    <div className="onboarding-header">
                        <div className="onboarding-icon-container">
                            <MapPin className="onboarding-icon" />
                        </div>
                        <h2>System Context</h2>
                        <p>Review mapped directories and environment. Remap anytime.</p>
                    </div>
                    <div className="onboarding-content">
                        {systemContext ? (
                            <>
                                <div className="debug-section">
                                    <h4>Directories</h4>
                                    <pre className="debug-text">{JSON.stringify(systemContext.directories || {}, null, 2)}</pre>
                                </div>
                                <div className="debug-section">
                                    <h4>Summary</h4>
                                    <pre className="debug-text">{JSON.stringify({
                                        applications: (systemContext.applications || []).length,
                                        processes: (systemContext.processes || []).length,
                    networkInterfaces: (systemContext.network?.interfaces || []).length,
                                        drives: (systemContext.filesystem?.drives || []).length,
                                        mappingDate: systemContext.mappingDate,
                                        source: systemContext.source,
                                    }, null, 2)}</pre>
                                </div>
                                <div className="debug-section">
                                    <h4>Raw Context JSON</h4>
                                    <pre className="debug-text">{JSON.stringify(systemContext, null, 2)}</pre>
                                </div>
                            </>
                        ) : (
                            <p className="no-commands">No system context available yet.</p>
                        )}
                    </div>
                    <div className="onboarding-actions">
                        <button onClick={() => setShowContextModal(false)} className="skip-button">Close</button>
                        <button onClick={copyContextToClipboard} className="start-mapping-button">Copy JSON</button>
                        <button onClick={handleRemap} className="start-mapping-button">Remap System</button>
                    </div>
                </div>
            </div>
        );
    };

    return (
        <div className="app">
            <div className="drag-region"></div>
            <div className="background"></div>
            
            <div className="main-container no-drag">
                {/* Header */}
                <header className="app-header">
                    <div className="header-content">
                        <div className="logo-section">
                            <img src={WSALogo} alt="WSA Logo" className="app-logo" />
                            <div className="logo-text">
                                <h1>Cypher</h1>
                                <p>Natural Language CLI</p>
                            </div>
                        </div>
                        <div className="header-controls">
                            <button 
                                onClick={() => setShowDebugPanel(!showDebugPanel)}
                                className={`debug-button ${showDebugPanel ? 'active' : ''}`}
                                title="Toggle Debug Panel"
                            >
                                <Bug className="button-icon" />
                            </button>
                            <button 
                                onClick={() => (systemContext ? setShowContextModal(true) : startOnboarding())}
                                className="remap-button"
                                title={systemContext ? "View / Remap System" : "Setup System"}
                            >
                                <MapPin className="button-icon" />
                            </button>
                        </div>
                    </div>
                </header>

                {/* Main Content */}
                <main className={`main-content ${isCompact ? 'compact' : 'spacious'}`}>
                    {/* Input Section */}
                    <section className="input-section">
                        <div className="input-card">
                            <div className="card-header">
                                <Zap className="card-icon" />
                                <h2>Describe Your Task</h2>
                            </div>
                            
                            <div className="input-area">
                                <textarea
                                    value={goal}
                                    onChange={(e) => setGoal(e.target.value)}
                                    onKeyDown={handleKeyPress}
                                    placeholder="What would you like me to automate? (e.g., organize my downloads folder, create a backup of documents, set up a development environment...)"
                                    className="task-input"
                                    rows={4}
                                    disabled={isExecuting}
                                />
                                
                                <div className="input-options">
                                    <div className="model-selector">
                                        <label htmlFor="model-select" className="model-label">
                                            <Settings className="model-icon" />
                                            Model:
                                        </label>
                                        <select
                                            id="model-select"
                                            value={selectedModel}
                                            onChange={(e) => handleModelChange(e.target.value)}
                                            disabled={isExecuting}
                                            className="model-select"
                                        >
                                            {availableModels.map((model) => (
                                                <option key={model} value={model}>
                                                    {model}
                                                </option>
                                            ))}
                                        </select>
                                    </div>

                                    <label className="vision-toggle">
                                        <input
                                            type="checkbox"
                                            checked={useVision}
                                            onChange={(e) => setUseVision(e.target.checked)}
                                            disabled={isExecuting}
                                        />
                                        <div className="toggle-indicator">
                                            <Eye className="toggle-icon" />
                                        </div>
                                        <span>Enable Vision Mode</span>
                                    </label>
                                </div>
                            </div>

                            <div className="action-buttons">
                                <button
                                    onClick={handleExecute}
                                    disabled={isExecuting || !goal.trim()}
                                    className="execute-button"
                                >
                                    {isExecuting ? (
                                        <>
                                            <Loader2 className="button-icon spinning" />
                                            Executing...
                                        </>
                                    ) : (
                                        <>
                                            <Play className="button-icon" />
                                            Execute Task
                                        </>
                                    )}
                                </button>
                                {isExecuting && (
                                    <button
                                        onClick={handleAbort}
                                        className="abort-button"
                                        title="Abort Task"
                                    >
                                        <X className="button-icon" />
                                        Abort
                                    </button>
                                )}
                            </div>
                        </div>
                    </section>

                    {/* Output Section */}
                    {(hasOutput || isExecuting) && (
                        <section className="output-section">
                            <div className="output-card">
                                <div className="card-header">
                                    <Terminal className="card-icon" />
                                    <h2>Execution Progress</h2>
                                    <div className="model-badge">
                                        <Settings className="model-badge-icon" />
                                        {selectedModel}
                                    </div>
                                </div>
                                
                                <div className="output-content">
                                    {isExecuting ? (
                                        <div className="execution-progress">
                                            {/* Progress Bar */}
                                            <div className="progress-section">
                                                <div className="progress-header">
                                                    <div className="progress-status" style={{ display: 'flex', alignItems: 'center', gap: 12, minHeight: 56 }}>
                                                        <JellyLoader numberOfCubes={6} width={120} height={36} cubeWidth={30} cubeHeight={20} dx={10} dy={-5} style={{ marginTop: 6 }} />
                                                        <span className="progress-text">{executionStage}</span>
                                                    </div>
                                                    <span className="progress-percentage">{progress}%</span>
                                                </div>
                                                <div className="progress-bar">
                                                    <div 
                                                        className="progress-fill" 
                                                        style={{ width: `${progress}%` }}
                                                    ></div>
                                                </div>
                                            </div>

                                            {/* Current Command */}
                                            {currentCommand && (
                                                <div className="current-command">
                                                    <div className="command-header">
                                                        <Command className="command-icon" />
                                                        <span>Executing Command</span>
                                                    </div>
                                                    <code className="command-text">{currentCommand}</code>
                                                </div>
                                            )}

                                            {/* Live Command History */}
                                            {commandHistory.length > 0 && (
                                                <div className="live-commands">
                                                    <h3>Command History</h3>
                                                    <div className="commands-list">
                                                        {commandHistory.map((cmd, index) => (
                                                            <div key={index} className="command-item completed">
                                                                <CheckCircle className="command-status-icon" />
                                                                <span className="command-number">{index + 1}</span>
                                                                <code className="command-code">{cmd}</code>
                                                            </div>
                                                        ))}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    ) : (
                                        <div className="results">
                                            {/* Final Results */}
                                            <div className="completion-status">
                                                <CheckCircle className="completion-icon" />
                                                <div className="completion-text">
                                                    <h3>Task Completed Successfully</h3>
                                                    <p>All commands executed without errors</p>
                                                </div>
                                            </div>

                                            <div className="result-message">{output}</div>
                                            
                                            {logs.length > 0 && (
                                                <div className="command-logs">
                                                    <h3>Final Command Sequence</h3>
                                                    <div className="logs-list">
                                                        {logs.map((log, index) => (
                                                            <div key={index} className="log-item">
                                                                <CheckCircle className="log-status-icon" />
                                                                <span className="log-number">{index + 1}</span>
                                                                <code className="log-command">{log}</code>
                                                            </div>
                                                        ))}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </div>
                            </div>
                        </section>
                    )}
                </main>

                {/* Debug Panel */}
                {showDebugPanel && (
                    <section className="debug-panel">
                        <div className="debug-header">
                            <Bug className="debug-icon" />
                            <h3>Live Debug Console</h3>
                            <button 
                                onClick={() => setShowDebugPanel(false)}
                                className="close-debug"
                            >
                                <X className="close-icon" />
                            </button>
                        </div>
                        <div className="debug-content">
                            <div className="debug-section">
                                <h4>System Context</h4>
                                <pre className="debug-text">
                                    {systemContext ? JSON.stringify(systemContext, null, 2) : 'No system context available'}
                                </pre>
                            </div>
                            {systemContext && systemContext.directories && (
                                <div className="debug-section">
                                    <h4>Mapped Directories</h4>
                                    <pre className="debug-text">{JSON.stringify(systemContext.directories, null, 2)}</pre>
                                </div>
                            )}
                            {systemContext && systemContext.applications && systemContext.applications.length > 0 && (
                                <div className="debug-section">
                                    <h4>Installed Applications</h4>
                                    <pre className="debug-text">{JSON.stringify(systemContext.applications.slice(0, 50), null, 2)}</pre>
                                </div>
                            )}
                            <div className="debug-section">
                                <h4>Live Commands</h4>
                                <div className="live-commands-list">
                                    {liveCommands.length > 0 ? (
                                        liveCommands.map((cmd, index) => (
                                            <div key={index} className="live-command">
                                                <span className="command-timestamp">{new Date(cmd.timestamp).toLocaleTimeString()}</span>
                                                <code className="command-text">{cmd.command}</code>
                                                <span className={`command-status ${cmd.status}`}>{cmd.status}</span>
                                            </div>
                                        ))
                                    ) : (
                                        <p className="no-commands">No live commands to display</p>
                                    )}
                                </div>
                            </div>
                        </div>
                    </section>
                )}

                {/* Footer */}
                <footer className="app-footer">
                    <p>Press <kbd>⌘ + Enter</kbd> to execute • Choose your AI model • Vision mode enables screen interaction</p>
                </footer>
            </div>
            
            {/* Onboarding Modal - Positioned outside main flow */}
            {isOnboarding && <OnboardingModal />}
            {showContextModal && <SystemContextModal />}
        </div>
    );
}

export default App;
