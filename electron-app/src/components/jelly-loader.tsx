'use client';

import React from 'react';
import { motion } from 'framer-motion';

// Props (untyped to avoid TS tooling):
// numberOfCubes, width, height, cubeWidth, cubeHeight, dx, dy, style
const JellyLoader = ({
    numberOfCubes = 8,
    width = 120,
    height = 48,
    cubeWidth = 40,
    cubeHeight = 24,
    dx = 10,
    dy = -6,
    style = {},
}) => {
    // Green palette to match the app
    const colors = [
        '#D1FAE5',
        '#A7F3D0',
        '#6EE7B7',
        '#34D399',
        '#10B981',
        '#059669',
        '#047857',
        '#065F46',
    ];

    const transition = {
        duration: 1.5,
        repeat: Infinity,
        repeatDelay: 0.5,
        ease: 'easeOut',
    };

    return (
        <div style={{ position: 'relative', width, height, ...style }}>
            {Array.from({ length: numberOfCubes }).map((_, index) => {
                const left = index * dx;
                const top = (height - cubeHeight) / 2 + index * dy;

                return (
                    <motion.span
                        key={index}
                        style={{
                            position: 'absolute',
                            height: cubeHeight,
                            width: cubeWidth,
                            borderRadius: 9999,
                            left,
                            top,
                            zIndex: numberOfCubes - index,
                            backgroundColor: colors[index % colors.length],
                            opacity: 1 - index * 0.05,
                        }}
                        initial={{ scale: 1 }}
                        animate={{ scale: [1, 0.75, 1], rotate: [0, 360] }}
                        transition={{
                            ...transition,
                            delay: index * 0.05,
                        }}
                    />
                );
            })}
        </div>
    );
};

export default JellyLoader;
