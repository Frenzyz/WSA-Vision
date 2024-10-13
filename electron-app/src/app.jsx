import React, { useState } from 'react';
import './styles.css';

function App() {
    const [goal, setGoal] = useState('');
    const [output, setOutput] = useState('');
    const [isExecuting, setIsExecuting] = useState(false);
    const [logs, setLogs] = useState([]);
    const [useVision, setUseVision] = useState(false);

    const handleSubmit = (e) => {
        e.preventDefault();

        if (goal.trim() === '') {
            alert('Please enter a goal.');
            return;
        }

        setIsExecuting(true);
        setOutput('Processing...');
        setLogs([]);

        fetch('http://localhost:8080/execute', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ goal, useVision }),
        })
            .then((response) => response.json())
            .then((data) => {
                setOutput(data.message);
                setLogs(data.logs);
                setIsExecuting(false);
            })
            .catch((error) => {
                console.error('Error:', error);
                setOutput('An error occurred. Check the console for details.');
                setIsExecuting(false);
            });
    };

    return (
        <div className="container">
            <h1><i className="fas fa-robot"></i> WSA Assistant</h1>
            <form onSubmit={handleSubmit} className="goal-form">
                <input
                    type="text"
                    value={goal}
                    onChange={(e) => setGoal(e.target.value)}
                    placeholder="Enter a goal"
                    disabled={isExecuting}
                />
                <label className="checkbox-container">
                    <input
                        type="checkbox"
                        checked={useVision}
                        onChange={(e) => setUseVision(e.target.checked)}
                        disabled={isExecuting}
                    />
                    Use Vision
                </label>
                <button type="submit" disabled={isExecuting}>
                    {isExecuting ? 'Executing...' : 'Submit'}
                </button>
            </form>
            <div className={`output ${isExecuting ? 'fade-in' : ''}`}>
                {isExecuting ? (
                    <div className="loader"></div>
                ) : (
                    <>
                        <div>{output}</div>
                        {logs.length > 0 && (
                            <div className="logs">
                                {logs.map((log, index) => (
                                    <div key={index}>{log}</div>
                                ))}
                            </div>
                        )}
                    </>
                )}
            </div>
        </div>
    );
}

export default App;
