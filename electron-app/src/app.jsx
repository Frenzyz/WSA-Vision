import React, { useState } from 'react';
import './styles.css';

function App() {
    const [goal, setGoal] = useState('');
    const [output, setOutput] = useState('');
    const [isExecuting, setIsExecuting] = useState(false);

    const handleSubmit = (e) => {
        e.preventDefault();

        if (goal.trim() === '') {
            alert('Please enter a goal.');
            return;
        }

        setIsExecuting(true);
        setOutput('Processing...');

        fetch('http://localhost:8080/execute', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ goal }),
        })
            .then((response) => response.json())
            .then((data) => {
                setOutput(data.message);
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
            <h1>WSA Assistant</h1>
            <form onSubmit={handleSubmit} className="goal-form">
                <input
                    type="text"
                    value={goal}
                    onChange={(e) => setGoal(e.target.value)}
                    placeholder="Enter a goal"
                    disabled={isExecuting}
                />
                <button type="submit" disabled={isExecuting}>
                    {isExecuting ? 'Executing...' : 'Submit'}
                </button>
            </form>
            <div className={`output ${isExecuting ? 'fade-in' : ''}`}>{output}</div>
        </div>
    );
}

export default App;
